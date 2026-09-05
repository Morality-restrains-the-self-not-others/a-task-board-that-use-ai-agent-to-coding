package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// preciseRestartFileName 登记文件相对仓库根的路径（.runall/ 已 gitignore，属运行时状态）。
const preciseRestartFileName = ".runall/precise_restart_services.txt"

// preciseRestartTTL 登记有效期：超过该时长视为过期登记，按钮置灰提示（OPT-20260807-026）。
const preciseRestartTTL = 24 * time.Hour

// RegistrationState 登记条目状态：pending 待处理 / failed 失败待重试（OPT-20260807-024）。
type RegistrationState string

const (
	RegistrationStatePending RegistrationState = "pending"
	RegistrationStateFailed  RegistrationState = "failed"
)

// RegistrationEntry 登记条目：服务名 + 状态 + 登记时间戳（unix 秒）。
// 文件行格式兼容旧版纯服务名（视为 pending、时间未知）；新版写
// `name\tstate\tunix_ts`（如 task-auth\tfailed\t1754611200）。
type RegistrationEntry struct {
	Name         string
	State        RegistrationState
	RegisteredAt int64
}

// absPreciseRestartPath 把登记路径收成绝对路径，避免 DEPLOY_MODE 下相对路径落在
// 编排器 cwd（clone-run 常见为 $DEPLOY_ROOT/runAll/）读到空文件。
func absPreciseRestartPath(p string) string {
	if strings.TrimSpace(p) == "" {
		return p
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	return abs
}

// preciseRestartFile 从主配置文件路径推导登记文件绝对路径。
// conf/runAll.yaml 与 runAll/config.yaml 的祖父目录均为仓库根（以 .ai.md 为标记）。
// DEPLOY_MODE=1 时读 $SOURCE_ROOT/.runall/（环境变量或 cutover.env，ADR-0056）。
// 可用环境变量 RUNALL_PRECISE_RESTART_FILE 覆盖。
func preciseRestartFile(cfgPath string) string {
	if env := strings.TrimSpace(os.Getenv("RUNALL_PRECISE_RESTART_FILE")); env != "" {
		return absPreciseRestartPath(env)
	}
	if deployModeActive() {
		if root := resolveSourceRoot(cfgPath); root != "" {
			return absPreciseRestartPath(filepath.Join(root, preciseRestartFileName))
		}
	}
	if cfgPath != "" {
		if abs, err := filepath.Abs(cfgPath); err == nil {
			root := filepath.Dir(filepath.Dir(abs))
			if fileExists(filepath.Join(root, ".ai.md")) {
				return absPreciseRestartPath(filepath.Join(root, preciseRestartFileName))
			}
		}
	}
	return absPreciseRestartPath(filepath.Join(".runall", "precise_restart_services.txt"))
}

// parseRegistrationEntry 解析单行登记：兼容旧版纯服务名（视为 pending、时间未知）
// 与新版 `name\tstate\tunix_ts`。返回零值表示该行不是有效登记（应跳过）。
func parseRegistrationEntry(line string) (RegistrationEntry, bool) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return RegistrationEntry{}, false
	}
	if idx := strings.Index(line, "#"); idx >= 0 {
		line = strings.TrimSpace(line[:idx])
		if line == "" {
			return RegistrationEntry{}, false
		}
	}
	parts := strings.Split(line, "\t")
	name := strings.TrimSpace(parts[0])
	if name == "" {
		return RegistrationEntry{}, false
	}
	entry := RegistrationEntry{Name: name, State: RegistrationStatePending}
	if len(parts) >= 3 {
		state := strings.TrimSpace(parts[1])
		if state == string(RegistrationStateFailed) {
			entry.State = RegistrationStateFailed
		}
		if ts, err := strconv.ParseInt(strings.TrimSpace(parts[2]), 10, 64); err == nil && ts > 0 {
			entry.RegisteredAt = ts
		}
	}
	return entry, true
}

// registrationEntryLine 序列化为一行（含状态与时间戳）。
func registrationEntryLine(e RegistrationEntry) string {
	return fmt.Sprintf("%s\t%s\t%d", e.Name, e.State, e.RegisteredAt)
}

// readRegisteredServices 读取登记文件，返回服务名（旧接口，保持调用方兼容）。
// 文件不存在视为空登记（返回空切片，无错误）。
func readRegisteredServices(path string) ([]string, error) {
	entries, err := readRegistrationEntries(path)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name)
	}
	return names, nil
}

// migrateLegacyRegistrationTimestamps 为无时间戳的旧格式行补当前 unix 秒，
// 并落盘为新格式（name\tstate\tts），避免纯名字行被误判为 TTL 过期。
func migrateLegacyRegistrationTimestamps(entries []RegistrationEntry, now int64) ([]RegistrationEntry, bool) {
	if now <= 0 {
		now = time.Now().Unix()
	}
	changed := false
	out := make([]RegistrationEntry, len(entries))
	for i, e := range entries {
		out[i] = e
		if e.RegisteredAt > 0 {
			continue
		}
		out[i].RegisteredAt = now
		if out[i].State == "" {
			out[i].State = RegistrationStatePending
		}
		changed = true
	}
	return out, changed
}

// readRegistrationEntries 读取登记文件为带状态与时间戳的条目，逐行解析，
// 忽略空行、整行 # 注释与行内 # 注释，保持顺序按名去重。
// 旧格式纯服务名行（RegisteredAt<=0）在读取时自动补戳并写回磁盘。
func readRegistrationEntries(path string) ([]RegistrationEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read precise-restart registrations %s: %w", path, err)
	}
	var seen = make(map[string]struct{})
	var result []RegistrationEntry
	sc := bufio.NewScanner(strings.NewReader(string(data)))
	for sc.Scan() {
		entry, ok := parseRegistrationEntry(sc.Text())
		if !ok {
			continue
		}
		if _, ok := seen[entry.Name]; ok {
			continue
		}
		seen[entry.Name] = struct{}{}
		result = append(result, entry)
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("parse precise-restart registrations %s: %w", path, err)
	}
	if len(result) > 0 {
		migrated, changed := migrateLegacyRegistrationTimestamps(result, time.Now().Unix())
		if changed {
			if err := writeRegistrationEntries(path, migrated); err != nil {
				return nil, err
			}
		}
		result = migrated
	}
	return result, nil
}

// writeRegisteredServices 全量写回登记文件（空切片即清空）。
// 旧接口：写服务名并附带 pending + 当前时间戳，避免再次产生无戳旧格式行。
func writeRegisteredServices(path string, names []string) error {
	now := time.Now().Unix()
	entries := make([]RegistrationEntry, 0, len(names))
	for _, n := range names {
		n = strings.TrimSpace(n)
		if n == "" {
			continue
		}
		entries = append(entries, RegistrationEntry{Name: n, State: RegistrationStatePending, RegisteredAt: now})
	}
	return writeRegistrationEntries(path, entries)
}

// writeRegistrationEntries 全量写回登记条目（空切片即清空）。
func writeRegistrationEntries(path string, entries []RegistrationEntry) error {
	var buf strings.Builder
	for _, e := range entries {
		buf.WriteString(registrationEntryLine(e))
		buf.WriteByte('\n')
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir precise-restart registrations %s: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(buf.String()), 0o644); err != nil {
		return fmt.Errorf("write precise-restart registrations %s: %w", path, err)
	}
	return nil
}

// appendRegisteredServices 追加登记并去重，返回合并后的完整列表。
// 新登记条目标记为 pending 并记录当前时间戳（OPT-20260807-026）。
func appendRegisteredServices(path string, names []string) ([]string, error) {
	existing, err := readRegistrationEntries(path)
	if err != nil {
		return nil, err
	}
	now := time.Now().Unix()
	merged := append([]RegistrationEntry(nil), existing...)
	for _, n := range names {
		n = strings.TrimSpace(n)
		if n == "" {
			continue
		}
		dup := false
		for i := range merged {
			if merged[i].Name == n {
				// 已存在：默认保留原状态与时间戳（含失败待重试项），不覆盖。
				// 例外：原登记已超 TTL 过期（OPT-20260807-026），重新登记视为恢复
				// 窗口——刷新时间戳并重置为 pending，使按钮不再置灰。
				if registrationExpired(merged[i], time.Now()) {
					merged[i].State = RegistrationStatePending
					merged[i].RegisteredAt = now
				}
				dup = true
				break
			}
		}
		if !dup {
			merged = append(merged, RegistrationEntry{Name: n, State: RegistrationStatePending, RegisteredAt: now})
		}
	}
	if err := writeRegistrationEntries(path, merged); err != nil {
		return nil, err
	}
	namesOut := make([]string, 0, len(merged))
	for _, e := range merged {
		namesOut = append(namesOut, e.Name)
	}
	return namesOut, nil
}

// registrationExpired 判断登记是否已超过有效期（默认 24h）。
// RegisteredAt<=0 的防御性过期：正常路径 readRegistrationEntries 会先自动补戳；
// 重复登记过期项时 appendRegisteredServices 会刷新时间戳并重置 pending。
func registrationExpired(e RegistrationEntry, now time.Time) bool {
	if e.RegisteredAt <= 0 {
		return true
	}
	return now.Sub(time.Unix(e.RegisteredAt, 0)) > preciseRestartTTL
}

// retainFailedEntries 将失败保留项写回登记文件：状态 failed、时间戳刷新为当前
// （保留原时间戳会让失败项随时间自然过期，重置便于重试窗口重算）。仅保留失败项。
func retainFailedEntries(path string, failed []string) error {
	now := time.Now().Unix()
	entries := make([]RegistrationEntry, 0, len(failed))
	for _, f := range failed {
		entries = append(entries, RegistrationEntry{Name: f, State: RegistrationStateFailed, RegisteredAt: now})
	}
	return writeRegistrationEntries(path, entries)
}

// preciseRestartConsumedAtFileName 水位线文件名：记录最近一次精准编译重启
// 「消费」登记的时间。自动扫描仅在脏文件 mtime 新于此水位时才重新登记，
// 避免「重启已清空 → Stop hook 见脏工作树 → 立刻重登全部服务」。
const preciseRestartConsumedAtFileName = "precise_restart_consumed_at"

// preciseRestartConsumedAtFile 由登记文件路径推导水位线路径（同目录）。
func preciseRestartConsumedAtFile(regPath string) string {
	dir := filepath.Dir(regPath)
	if dir == "" || dir == "." {
		return filepath.Join(".runall", preciseRestartConsumedAtFileName)
	}
	return filepath.Join(dir, preciseRestartConsumedAtFileName)
}

// writePreciseRestartConsumedAt 写入消费水位线（unix 秒）。
func writePreciseRestartConsumedAt(regPath string, ts int64) error {
	if ts <= 0 {
		ts = time.Now().Unix()
	}
	wm := preciseRestartConsumedAtFile(regPath)
	if err := os.MkdirAll(filepath.Dir(wm), 0o755); err != nil {
		return fmt.Errorf("mkdir precise-restart consumed-at %s: %w", filepath.Dir(wm), err)
	}
	if err := os.WriteFile(wm, []byte(strconv.FormatInt(ts, 10)+"\n"), 0o644); err != nil {
		return fmt.Errorf("write precise-restart consumed-at %s: %w", wm, err)
	}
	return nil
}

// rewriteRegistrationsAfterRun 成功收尾重写登记文件：
//   - 本批失败项 → failed（刷新时间戳）；
//   - 本批成功项 → 删除；
//   - 运行期间并发追加、且不属于本批 original 的项 → 保留 pending。
//
// 返回最终保留的服务名列表。
func rewriteRegistrationsAfterRun(path string, original map[string]bool, failed, succeeded []string) ([]string, error) {
	now := time.Now().Unix()
	failedSet := make(map[string]bool, len(failed))
	for _, f := range failed {
		f = strings.TrimSpace(f)
		if f != "" {
			failedSet[f] = true
		}
	}
	succeededSet := make(map[string]bool, len(succeeded))
	for _, s := range succeeded {
		s = strings.TrimSpace(s)
		if s != "" {
			succeededSet[s] = true
		}
	}

	current, err := readRegistrationEntries(path)
	if err != nil {
		return nil, err
	}

	var keep []RegistrationEntry
	seen := make(map[string]bool)

	// 1) 本批失败项优先（状态 failed）。
	for _, f := range failed {
		f = strings.TrimSpace(f)
		if f == "" || seen[f] {
			continue
		}
		seen[f] = true
		keep = append(keep, RegistrationEntry{Name: f, State: RegistrationStateFailed, RegisteredAt: now})
	}

	// 2) 当前文件中：非本批 original、非成功、非已记失败 → 并发新登记，保留。
	for _, e := range current {
		name := strings.TrimSpace(e.Name)
		if name == "" || seen[name] || succeededSet[name] || failedSet[name] {
			continue
		}
		if original[name] {
			// 本批成员若既非成功也非失败，理论上不应出现于正常收尾；丢弃以免脏 pending。
			continue
		}
		seen[name] = true
		state := e.State
		if state == "" {
			state = RegistrationStatePending
		}
		ts := e.RegisteredAt
		if ts <= 0 {
			ts = now
		}
		keep = append(keep, RegistrationEntry{Name: name, State: state, RegisteredAt: ts})
	}

	if err := writeRegistrationEntries(path, keep); err != nil {
		return nil, err
	}
	names := make([]string, 0, len(keep))
	for _, e := range keep {
		names = append(names, e.Name)
	}
	return names, nil
}

// keepEntriesOnAbort 中断时保留：未处理 pending + 失败 failed；已成功项丢弃。
func keepEntriesOnAbort(succeeded, pending, failed []string, originalTs map[string]int64, now int64) []RegistrationEntry {
	if now <= 0 {
		now = time.Now().Unix()
	}
	succeededSet := make(map[string]bool, len(succeeded))
	for _, s := range succeeded {
		succeededSet[s] = true
	}
	var keep []RegistrationEntry
	seen := make(map[string]bool)
	for _, name := range pending {
		name = strings.TrimSpace(name)
		if name == "" || seen[name] || succeededSet[name] {
			continue
		}
		seen[name] = true
		ts := originalTs[name]
		keep = append(keep, RegistrationEntry{Name: name, State: RegistrationStatePending, RegisteredAt: ts})
	}
	for _, name := range failed {
		name = strings.TrimSpace(name)
		if name == "" || seen[name] || succeededSet[name] {
			continue
		}
		seen[name] = true
		keep = append(keep, RegistrationEntry{Name: name, State: RegistrationStateFailed, RegisteredAt: now})
	}
	return keep
}

// clearRegisteredServices 清空登记文件。
func clearRegisteredServices(path string) error {
	return writeRegisteredServices(path, nil)
}

// clearPreciseRestartRegistrationsAfterFullRebuild 页头「全部重新编译」正常完成后
// 视为已消费精准编译登记：清空登记文件并写入 consumed-at 水位线。
// 中断取消路径不得调用。
func clearPreciseRestartRegistrationsAfterFullRebuild(cfgPath string) error {
	path := preciseRestartFile(cfgPath)
	if err := clearRegisteredServices(path); err != nil {
		return err
	}
	return writePreciseRestartConsumedAt(path, time.Now().Unix())
}

// trimRegistrationsAfterGroupBuild 分组编译成功后按组裁剪登记（OPT-20260811-036）：
// 仅移除本组内已登记且编译成功的服务名；失败/跳过项、本组外服务与运行期间并发追加的
// 新登记一律保留。全量 BuildAll 仍走清空登记；分组编译只做局部修剪，避免误消其它组待重启项。
// built 为空（本组无一编译成功）时为 no-op。
func trimRegistrationsAfterGroupBuild(path string, built []string) error {
	builtSet := make(map[string]bool, len(built))
	for _, b := range built {
		b = strings.TrimSpace(b)
		if b != "" {
			builtSet[b] = true
		}
	}
	if len(builtSet) == 0 {
		return nil
	}
	current, err := readRegistrationEntries(path)
	if err != nil {
		return err
	}
	keep := make([]RegistrationEntry, 0, len(current))
	for _, e := range current {
		if builtSet[strings.TrimSpace(e.Name)] {
			continue
		}
		keep = append(keep, e)
	}
	return writeRegistrationEntries(path, keep)
}

// resolveRegisteredServices 将登记名解析为配置中的服务列表：
//  1. 精确匹配 runAll.yaml 服务名（如 task-auth、saas-backend）→ 单元素；
//  2. 匹配 conf_app 值（如 django）→ 单元素；
//  3. 匹配工作目录名（如 taskAuth、taskBill；taskFE/app → 首段 taskFE）。
//     同一 working_dir 挂多个服务时（如 taskEvents → 全部 task-events-*）全部展开。
//
// 返回空切片表示无法解析。
func (r *Runner) resolveRegisteredServices(name string) []*Service {
	if r == nil || r.cfg == nil {
		return nil
	}
	target := strings.TrimSpace(name)
	if target == "" {
		return nil
	}
	var byWorkingDir []*Service
	for gi := range r.cfg.Groups {
		for si := range r.cfg.Groups[gi].Services {
			svc := &r.cfg.Groups[gi].Services[si]
			if svc.Name == target || svc.ConfApp == target {
				return []*Service{svc}
			}
			if workingDirAliasSeg(r, svc.WorkingDir) == target {
				byWorkingDir = append(byWorkingDir, svc)
			}
		}
	}
	if len(byWorkingDir) == 0 {
		return nil
	}
	return byWorkingDir
}

// resolveRegisteredService 单值解析（仅唯一匹配时成功）。
// 多服务共享 working_dir 别名（如 taskEvents）请用 resolveRegisteredServices 展开。
func (r *Runner) resolveRegisteredService(name string) *Service {
	svcs := r.resolveRegisteredServices(name)
	if len(svcs) == 1 {
		return svcs[0]
	}
	return nil
}

// workingDirAliasSeg 返回工作目录的相对首段别名（taskFE/app → taskFE）。
// 配置加载时 resolveWorkingDirs（config.go）已将相对 working_dir 绝对化为
// <monorepoRoot>/taskFE/app —— 直接按 "/" 切分绝对路径首段会得到空串，导致
// 按工作目录名登记的别名（如 taskFE）永远解析失败（"unknown service"）。
// 因此先剥掉仓库根前缀再取首段；相对路径（单测直接构造）原样取首段；
// 仓库根不可识别时退化为剥前导斜杠取首段（兼容 /repo/taskFE/app 常见布局）。
func workingDirAliasSeg(r *Runner, dir string) string {
	d := strings.TrimSpace(dir)
	if d == "" {
		return ""
	}
	slashed := filepath.ToSlash(d)
	if !strings.HasPrefix(slashed, "/") {
		return strings.SplitN(slashed, "/", 2)[0]
	}
	if r != nil {
		if root, err := r.monorepoRoot(); err == nil && root != "" {
			if rel, err := filepath.Rel(root, d); err == nil && rel != "." && !strings.HasPrefix(rel, "..") {
				return strings.SplitN(filepath.ToSlash(rel), "/", 2)[0]
			}
		}
	}
	return strings.SplitN(strings.TrimPrefix(slashed, "/"), "/", 2)[0]
}
