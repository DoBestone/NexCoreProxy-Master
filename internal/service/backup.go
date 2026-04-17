package service

import (
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// BackupService 数据库备份
//
// 策略：
//   - 每日 02:00 跑 mysqldump，输出到 BackupDir/<dbname>-YYYYMMDD-HHMMSS.sql.gz
//   - 保留：最近 7 天日备 + 4 个周备（每周日的）+ 12 个月备（每月 1 号的）
//   - 不上传到 S3：留作 Phase 2 扩展（避免引入 minio-go 等大依赖）
//
// 配置通过运行时 + 环境变量注入：
//   BACKUP_DIR=/var/lib/nexcore-master/backups
//   BACKUP_MYSQLDUMP_BIN=mysqldump
type BackupService struct {
	BackupDir   string
	MysqlDump   string
	DBHost, DBPort, DBUser, DBPass, DBName string
}

func NewBackupService() *BackupService {
	dir := os.Getenv("BACKUP_DIR")
	if dir == "" {
		dir = "/var/lib/nexcore-master/backups"
	}
	bin := os.Getenv("BACKUP_MYSQLDUMP_BIN")
	if bin == "" {
		bin = "mysqldump"
	}
	return &BackupService{BackupDir: dir, MysqlDump: bin}
}

// AttachDBConfig 由 main 注入连接参数
func (s *BackupService) AttachDBConfig(host, port, user, pass, name string) {
	s.DBHost, s.DBPort, s.DBUser, s.DBPass, s.DBName = host, port, user, pass, name
}

// RunOnce 跑一次备份；通常由 cron 每天调一次
func (s *BackupService) RunOnce() {
	if s.DBName == "" {
		log.Printf("[backup] db config not attached, skip")
		return
	}
	if err := os.MkdirAll(s.BackupDir, 0o750); err != nil {
		log.Printf("[backup] mkdir %s failed: %v", s.BackupDir, err)
		return
	}
	stamp := time.Now().Format("20060102-150405")
	target := filepath.Join(s.BackupDir, fmt.Sprintf("%s-%s.sql.gz", s.DBName, stamp))

	// mysqldump | gzip > target
	args := []string{
		"-h", s.DBHost, "-P", s.DBPort,
		"-u", s.DBUser, "--password=" + s.DBPass,
		"--single-transaction", "--quick", "--routines",
		"--default-character-set=utf8mb4",
		s.DBName,
	}
	cmd := exec.Command(s.MysqlDump, args...)

	out, err := os.Create(target)
	if err != nil {
		log.Printf("[backup] create file failed: %v", err)
		return
	}
	defer out.Close()
	gz := gzip.NewWriter(out)
	defer gz.Close()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("[backup] stdout pipe failed: %v", err)
		return
	}
	if err := cmd.Start(); err != nil {
		log.Printf("[backup] start mysqldump failed: %v", err)
		return
	}
	if _, err := io.Copy(gz, stdout); err != nil {
		log.Printf("[backup] copy failed: %v", err)
		_ = cmd.Wait()
		return
	}
	if err := cmd.Wait(); err != nil {
		log.Printf("[backup] mysqldump exit failed: %v", err)
		_ = os.Remove(target)
		return
	}

	info, _ := os.Stat(target)
	size := int64(0)
	if info != nil {
		size = info.Size()
	}
	log.Printf("[backup] saved %s (%d bytes)", target, size)

	s.cleanup()
}

// cleanup 按"日 7 / 周 4 / 月 12"策略保留
//
// 简单实现：按日期排序，标记应保留的文件名集合，剩余删除。
func (s *BackupService) cleanup() {
	entries, err := os.ReadDir(s.BackupDir)
	if err != nil {
		return
	}
	type fileInfo struct {
		name string
		date time.Time
	}
	var files []fileInfo
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".sql.gz") {
			continue
		}
		// 文件名格式 <db>-YYYYMMDD-HHMMSS.sql.gz
		parts := strings.Split(strings.TrimSuffix(e.Name(), ".sql.gz"), "-")
		if len(parts) < 3 {
			continue
		}
		ts := parts[len(parts)-2] + parts[len(parts)-1]
		t, err := time.Parse("20060102150405", ts)
		if err != nil {
			continue
		}
		files = append(files, fileInfo{e.Name(), t})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].date.After(files[j].date) })

	keep := make(map[string]struct{})
	dailyCount, weeklyCount, monthlyCount := 0, 0, 0
	for _, f := range files {
		// 日备：保留前 7 个
		if dailyCount < 7 {
			keep[f.name] = struct{}{}
			dailyCount++
		}
		// 周备：每周日的备份保留 4 个
		if f.date.Weekday() == time.Sunday && weeklyCount < 4 {
			keep[f.name] = struct{}{}
			weeklyCount++
		}
		// 月备：每月 1 号的保留 12 个
		if f.date.Day() == 1 && monthlyCount < 12 {
			keep[f.name] = struct{}{}
			monthlyCount++
		}
	}

	for _, f := range files {
		if _, ok := keep[f.name]; ok {
			continue
		}
		_ = os.Remove(filepath.Join(s.BackupDir, f.name))
	}
}
