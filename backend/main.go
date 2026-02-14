package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

// --- 結構定義 (與前端的 JSON 格式對應) ---

// 單一課程紀錄 (從學生上傳的 JSON 中解析出來的扁平化結構)
type StudentCourse struct {
	Name         string  `json:"name"`
	Credit       float64 `json:"credit"`
	Score        string  `json:"score"` // 可能是數字或 "成績未到或無成績"
	IsInProgress bool    `json:"isInProgress"`
	IsPassed     bool    `json:"isPassed"`
	Semester     string  `json:"semester"`
	IsCapped     bool    `json:"isCapped"`
}

// 學程要求中的一個分類
type ProgramRequirement struct {
	Category   string   `json:"category"`
	MinCount   int      `json:"min_count"`
	MaxCount   int      `json:"max_count"`
	MaxCredits float64  `json:"max_credits"` // 該分類最高認列學分
	MinCredits float64  `json:"min_credits"`
	Courses    []string `json:"courses"` // 課程名稱列表
}

// 單一學程定義
type Program struct {
	Name                    string               `json:"name"`
	MinCredits              float64              `json:"min_credits"`
	Description             string               `json:"description"`
	Requirements            []ProgramRequirement `json:"requirements"`
	Type                    string               `json:"type"`                      // "micro" (微學程) or "credit" (學分學程)
	GeneralEducationCourses []string             `json:"general_education_courses"` // 通識課程列表 (全域限修一門)
}

// 檢核結果中的一個分類結果
type CategoryResult struct {
	Category        string          `json:"category"`
	RequiredCount   int             `json:"requiredCount"`
	RequiredCredits float64         `json:"requiredCredits"`
	PassedCount     int             `json:"passedCount"`
	PassedCredits   float64         `json:"passedCredits"`
	IsMet           bool            `json:"isMet"`
	PassedCourses   []StudentCourse `json:"passedCourses"`
	LimitExceeded   bool            `json:"limitExceeded"`
	ExceededMessage string          `json:"exceededMessage"`
}

// 最終檢核結果
type CheckResult struct {
	ProgramName        string           `json:"programName"`
	IsCompleted        bool             `json:"isCompleted"`
	TotalPassedCredits string           `json:"totalPassedCredits"` // 傳回字串方便前端顯示
	MinRequiredCredits string           `json:"minRequiredCredits"`
	TotalCreditsMet    bool             `json:"totalCreditsMet"`
	AllCategoriesMet   bool             `json:"allCategoriesMet"`
	CategoryResults    []CategoryResult `json:"categoryResults"`
	InProgressCourses  []StudentCourse  `json:"inProgressCourses"`
	ProgramDescription string           `json:"programDescription"`
	AvgScoreRequired   bool             `json:"avgScoreRequired"`   // 是否需要檢核平均成績
	AvgScore           string           `json:"avgScore"`           // 平均成績
	AvgScoreMet        bool             `json:"avgScoreMet"`        // 平均成績是否達標
	AvgScoreThreshold  string           `json:"avgScoreThreshold"`  // 平均成績門檻
	RestrictionMessage string           `json:"restrictionMessage"` // 資格限制訊息
}

// 輔助結構：用於匹配單一學年/學期的紀錄
type GradeRecordList struct {
	AcademicYear string `json:"AcademicYear"`
	GradeRecords []struct {
		CourseName string `json:"courseName"`
		Credit     string `json:"credit"`
		Score      string `json:"score"`
		// ... 其他欄位
		AcademicYear string `json:"academicYear"`
		Semester     string `json:"semester"`
	} `json:"GradeRecords"`
}

// 輔助結構：用於匹配 '課業學習' 欄位
type AcademicInfo struct {
	GradeRecordList []GradeRecordList `json:"gradeRecordList"`
	AboutMe         struct {
		RegisterMajor string `json:"registerMajor"`
	} `json:"aboutMe"`
}

// 頂層結構：用於匹配 JSON 檔案外層的陣列
type StudentDataWrapper []struct {
	AcademicInfo AcademicInfo `json:"課業學習"`
}

// 推薦結果結構
type Recommendation struct {
	ProgramID           string           `json:"programID"`
	ProgramName         string           `json:"programName"`
	Type                string           `json:"type"`
	TotalPassedCredits  float64          `json:"totalPassedCredits"`
	MinCredits          float64          `json:"minCredits"`
	PassedPrereqCredits float64          `json:"passedPrereqCredits"` // 新增：已修先修學分
	CompletionRate      float64          `json:"completionRate"`
	IsCompleted         bool             `json:"isCompleted"`
	IsRestricted        bool             `json:"isRestricted"`
	CategoryResults     []CategoryResult `json:"categoryResults"`
}

// --- 全局變數 ---
var programs map[string]Program
var programsByCollege map[string]map[string]Program
var businessMajors map[string]bool

// --- 輔助函式 ---

// 檢查分數是否及格 (Go 實作)
func isPassed(scoreStr string) bool {
	const PASSING_SCORE = 60.0
	score, err := strconv.ParseFloat(scoreStr, 64)
	return err == nil && score >= PASSING_SCORE
}

// 檢查是否修習中
func isInProgress(scoreStr string) bool {
	return strings.TrimSpace(scoreStr) == "成績未到或無成績"
}

// 載入學程定義
func loadPrograms() error {
	programsByCollege = make(map[string]map[string]Program)
	programs = make(map[string]Program)

	// 定義檔案與學程類型的對應
	files := map[string]string{
		"data/micro_programs.json":              "micro",
		"data/credit_programs.json":             "credit",
		"data/commerce_specialty_programs.json": "specialty",
	}

	for filename, pType := range files {
		file, err := os.ReadFile(filename)
		if err != nil {
			return err
		}

		var currentFilePrograms map[string]map[string]Program
		err = json.Unmarshal(file, &currentFilePrograms)
		if err != nil {
			return fmt.Errorf("無法解析 %s: %w", filename, err)
		}

		for college, collegePrograms := range currentFilePrograms {
			if _, ok := programsByCollege[college]; !ok {
				programsByCollege[college] = make(map[string]Program)
			}
			for id, p := range collegePrograms {
				p.Type = pType // 標記學程類型
				programsByCollege[college][id] = p
				programs[id] = p
			}
		}
	}

	// Post-process "跨院" (Interdisciplinary) programs
	if interdisciplinary, ok := programsByCollege["跨院"]; ok {
		delete(programsByCollege, "跨院") // Remove the category

		for id, p := range interdisciplinary {
			originalName := p.Name
			start := strings.LastIndex(originalName, "（")
			end := strings.LastIndex(originalName, "）")

			if start != -1 && end != -1 && end > start {
				collegesPart := originalName[start+len("（") : end]
				colleges := strings.Split(collegesPart, " x ")

				p.Name = originalName[:start]
				programs[id] = p // Update global map

				for _, college := range colleges {
					college = strings.TrimSpace(college)
					if _, exists := programsByCollege[college]; !exists {
						programsByCollege[college] = make(map[string]Program)
					}
					programsByCollege[college][id] = p
				}
			}
		}
	}
	return nil
}

// 載入系所分類資料 (用於判斷商學院學生)
func loadDepartments() error {
	file, err := os.ReadFile("data/departments_grouped.json")
	if err != nil {
		return err
	}

	var groups map[string]struct {
		CategoryName string `json:"category_name"`
		Departments  []struct {
			Name string `json:"name"`
		} `json:"departments"`
	}
	if err := json.Unmarshal(file, &groups); err != nil {
		return fmt.Errorf("無法解析 departments_grouped.json: %w", err)
	}

	businessMajors = make(map[string]bool)
	if group, ok := groups["3"]; ok { // "3" 代表商學院
		for _, dept := range group.Departments {
			businessMajors[dept.Name] = true
		}
	}
	return nil
}

// 解析並扁平化學生的歷年成績資料。
func loadStudentData(data []byte) ([]StudentCourse, string, error) {
	var rawData StudentDataWrapper

	if err := json.Unmarshal(data, &rawData); err != nil {
		return nil, "", fmt.Errorf("解析頂層 JSON 結構失敗: %w", err)
	}

	if len(rawData) == 0 || len(rawData[0].AcademicInfo.GradeRecordList) == 0 {
		return nil, "", fmt.Errorf("JSON 結構不符預期或未找到課程紀錄")
	}

	flatCourses := []StudentCourse{}

	// 進入 gradeRecordList
	gradeRecordList := rawData[0].AcademicInfo.GradeRecordList

	for _, academicYearRecord := range gradeRecordList {
		if len(academicYearRecord.GradeRecords) > 0 {
			for _, course := range academicYearRecord.GradeRecords {
				// 確保所有字串都被清理
				courseName := strings.TrimSpace(course.CourseName)
				scoreStr := strings.TrimSpace(course.Score)
				creditStr := strings.TrimSpace(course.Credit)
				semesterStr := fmt.Sprintf("%s-%s", strings.TrimSpace(course.AcademicYear), strings.TrimSpace(course.Semester))

				credit, _ := strconv.ParseFloat(creditStr, 64)

				flatCourses = append(flatCourses, StudentCourse{
					Name:         courseName,
					Credit:       credit,
					Score:        scoreStr,
					IsPassed:     isPassed(scoreStr),
					IsInProgress: isInProgress(scoreStr),
					Semester:     semesterStr,
				})
			}
		}
	}

	if len(flatCourses) == 0 {
		return nil, "", fmt.Errorf("檔案解析成功，但未找到有效的課程紀錄")
	}

	major := rawData[0].AcademicInfo.AboutMe.RegisterMajor

	return flatCourses, major, nil
}

// 核心檢核邏輯 (與原 JS checkProgramCompletion 邏輯對應)
// 檢核學生課程是否符合指定學分學程的要求。
// 注意：本函式依賴於全局變數 `programs`
func checkProgramCompletion(programID string, courses []StudentCourse, studentMajor string) CheckResult {
	program, ok := programs[programID]
	if !ok {
		// 如果學程 ID 無效，回傳一個錯誤結果
		return CheckResult{ProgramName: fmt.Sprintf("學程 ID %s 不存在", programID)}
	}

	// 階段 1: 預處理學程要求
	localRequirements, programCourseNamesClean, geCourseNames, courseInstructorMap := preprocessRequirements(programID, program)

	// 階段 2: 篩選並處理課程
	completedCourses, inProgressCourses := filterAndProcessCourses(programID, courses, &localRequirements, programCourseNamesClean, geCourseNames, courseInstructorMap)

	// 檢查是否有通識課程超限 (用於後續顯示)
	geLimitExceeded := false
	if len(program.GeneralEducationCourses) > 0 {
		geCount := 0
		for _, c := range completedCourses {
			if geCourseNames[c.Name] {
				geCount++
			}
		}
		// filterAndProcessCourses 已經將多餘的通識課程移除，所以這裡我們比較原始數量
		// 但因為我們沒有保留原始的 relevantPassed，這裡可以用一個簡單的邏輯：
		// 如果 filterAndProcessCourses 已經處理了，我們其實不需要知道是否超限，除非要顯示警告。
		// 為了顯示警告，我們可以在這裡重新檢查原始輸入中符合通識的數量。
		rawGeCount := 0
		for _, c := range courses {
			if geCourseNames[c.Name] && c.IsPassed {
				rawGeCount++
			}
		}
		if rawGeCount > 1 {
			geLimitExceeded = true
		}
	}

	totalPassedCredits := 0.0 // 將由後續計算有效學分決定

	// 步驟 2 & 3: 檢核分類要求 (門數) 和總學分
	categoryResults := []CategoryResult{}

	// 特殊處理：管理會計專業學程 (management_accounting)
	isManagementAccounting := programID == "management_accounting"

	if isManagementAccounting {
		var isMet bool
		categoryResults, isMet, totalPassedCredits = processManagementAccounting(program, completedCourses)
		// allCategoriesMet 將在 postprocessResults 中統一計算
		_ = isMet
	} else {
		// 一般學程邏輯：呼叫 special_handlers.go 中的函式
		categoryResults, totalPassedCredits = processStandardRequirements(localRequirements, completedCourses)

		// 如果有通識課程超限，加入一個額外的分類結果顯示
		if len(program.GeneralEducationCourses) > 0 && geLimitExceeded {
			// 找出被採計的通識課程 (在 filterAndProcessCourses 中已處理為僅剩一門或零門)
			var gePassed []StudentCourse
			for _, c := range completedCourses {
				if geCourseNames[c.Name] {
					gePassed = append(gePassed, c)
				}
			}
			geCredits := 0.0
			for _, c := range gePassed {
				geCredits += c.Credit
			}
			categoryResults = append(categoryResults, CategoryResult{
				Category:        "通識課程 (全域限制)",
				RequiredCount:   1, // 顯示限制
				PassedCount:     len(gePassed),
				PassedCredits:   geCredits,
				IsMet:           true,
				PassedCourses:   gePassed,
				LimitExceeded:   true,
				ExceededMessage: "通識課程認列以一門為限 (已自動採計最高分者)",
			})
		}
	}

	// 階段 3: 後處理 (跨群檢核、平均成績、系所限制等)
	categoryResults, allCategoriesMet, restrictionMessage, avgScoreRequired, avgScoreStr, avgScoreMet, avgScoreThreshold, totalPassedCredits := postprocessResults(programID, program, studentMajor, categoryResults, totalPassedCredits)

	// 步驟 4: 總結
	totalCreditsMet := totalPassedCredits >= program.MinCredits
	isCompleted := totalCreditsMet && allCategoriesMet

	// 若有平均成績要求且未達標，則視為未修畢
	if avgScoreRequired && !avgScoreMet {
		isCompleted = false
	}

	return CheckResult{
		ProgramName:        program.Name,
		IsCompleted:        isCompleted,
		TotalPassedCredits: fmt.Sprintf("%.1f", totalPassedCredits),
		MinRequiredCredits: fmt.Sprintf("%.1f", program.MinCredits),
		TotalCreditsMet:    totalCreditsMet,
		AllCategoriesMet:   allCategoriesMet,
		CategoryResults:    categoryResults,
		InProgressCourses:  inProgressCourses,
		ProgramDescription: program.Description,
		AvgScoreRequired:   avgScoreRequired,
		AvgScore:           avgScoreStr,
		AvgScoreMet:        avgScoreMet,
		AvgScoreThreshold:  avgScoreThreshold,
		RestrictionMessage: restrictionMessage,
	}
}

// --- HTTP 處理函式 ---

// 用於防止休眠的健康檢查
func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// 獲取學程列表
func getPrograms(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(programsByCollege)
}

// 輔助函式：從請求中解析學生資料
func parseStudentDataFromRequest(r *http.Request) ([]StudentCourse, string, error) {
	// 1. 解析 multipart 表單
	err := r.ParseMultipartForm(32 << 20) // 32MB
	if err != nil {
		return nil, "", fmt.Errorf("解析表單失敗: %w", err)
	}

	// 2. 讀取學生 JSON 檔案
	file, _, err := r.FormFile("student_json")
	if err != nil {
		return nil, "", fmt.Errorf("讀取檔案失敗: %w", err)
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return nil, "", fmt.Errorf("讀取檔案內容失敗: %w", err)
	}

	// 3. 解析學生課程資料
	return loadStudentData(fileBytes)
}

// 處理檔案上傳和檢核
func checkProgramsHandler(w http.ResponseWriter, r *http.Request) {
	// 處理 OPTIONS 請求 (CORS 預檢)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// 解析學生資料
	studentCourses, major, err := parseStudentDataFromRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 獲取選取的學程 ID
	programIDsStr := r.PostFormValue("program_ids")
	if programIDsStr == "" {
		http.Error(w, "請選取至少一個學程 ID", http.StatusBadRequest)
		return
	}
	// 假設前端傳送的是逗號分隔的 ID 字串
	programIDs := strings.Split(programIDsStr, ",")

	// 執行檢核
	var results []CheckResult
	for _, id := range programIDs {
		result := checkProgramCompletion(id, studentCourses, major)
		results = append(results, result)
	}

	// 回傳結果
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// 處理學程推薦 (遍歷所有學程並回傳符合一定程度者)
func recommendProgramsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// 解析學生資料
	studentCourses, major, err := parseStudentDataFromRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 遍歷所有學程進行檢核
	var recommendations []Recommendation

	for id, program := range programs {
		isRestricted := false
		// 若為外語專長商管學分學程且學生為商學院學生，則排除於推薦列表
		if id == "foreign_language_student_business_primer" && businessMajors[major] {
			isRestricted = true
		}
		// 若為管理會計專業學程且學生非商學院學生，則排除於推薦列表
		if id == "management_accounting" && !businessMajors[major] {
			isRestricted = true
		}

		result := checkProgramCompletion(id, studentCourses, major)

		passed, _ := strconv.ParseFloat(result.TotalPassedCredits, 64)
		min, _ := strconv.ParseFloat(result.MinRequiredCredits, 64)

		// 計算先修課程學分 (用於加分與計算百分比，但不計入 TotalPassedCredits)
		passedPrereq := 0.0
		requiredPrereq := 0.0

		for _, cat := range result.CategoryResults {
			if strings.HasPrefix(cat.Category, "先修課程") {
				passedPrereq += cat.PassedCredits
				requiredPrereq += cat.RequiredCredits
			}
		}

		// 計算完成度：(主學程已修 + 先修已修) / (主學程應修 + 先修應修)
		totalPassed := passed + passedPrereq
		totalRequired := min + requiredPrereq

		rate := 0.0
		if totalRequired > 0 {
			rate = totalPassed / totalRequired
		}

		// 推薦門檻：完成度達 20% 以上 (避免僅修一門通識就推薦所有學程)
		if rate >= 0.2 {
			recommendations = append(recommendations, Recommendation{
				ProgramID:           id,
				ProgramName:         program.Name,
				Type:                program.Type,
				TotalPassedCredits:  passed,
				MinCredits:          min,
				PassedPrereqCredits: passedPrereq,
				CompletionRate:      rate,
				IsCompleted:         result.IsCompleted,
				IsRestricted:        isRestricted,
				CategoryResults:     result.CategoryResults,
			})
		}
	}

	// 排序：依完成度由高至低
	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].CompletionRate > recommendations[j].CompletionRate
	})

	// 6. 篩選前五名 (包含並列)
	var topRecommendations []Recommendation
	if len(recommendations) > 0 {
		currentRank := 1
		lastRate := recommendations[0].CompletionRate

		for _, rec := range recommendations {
			// 若完成度小於上一筆，則名次遞增
			if rec.CompletionRate < lastRate {
				currentRank++
				lastRate = rec.CompletionRate
			}

			// 若名次已超過 5，則停止加入
			if currentRank > 5 {
				break
			}

			topRecommendations = append(topRecommendations, rec)
		}
	}

	// 回傳結果
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(topRecommendations)
}

// 輔助函式：標準化課程名稱，確保比對準確
func normalizeCourseName(name string) string {
	return strings.TrimSpace(name)
}

// 簡單的 CORS 中間件範例
func commonMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Access-Control-Allow-Origin", "*") // 生產環境建議指定前端網址
		w.Header().Add("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Add("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	// 1. 處理 Port：優先讀取環境變數 PORT，若無則預設為 10000 (Render 常用) 或 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// 2. 讀取 micro_programs.json (假設你在 init 或 main 中讀取)
	// 如果你依照之前的建議使用 "cd backend && ./main" 啟動
	// 程式就能直接透過 "micro_programs.json" 讀取到檔案
	err := loadPrograms()
	if err != nil {
		fmt.Printf("初始化失敗: %v\n", err)
		os.Exit(1)
	}
	// 載入系所資料
	err = loadDepartments()
	if err != nil {
		fmt.Printf("載入系所資料失敗: %v\n", err)
		os.Exit(1)
	}

	r := mux.NewRouter()
	// ... 你的路由設定 ...

	r.HandleFunc("/healthcheck", healthCheckHandler).Methods("GET")
	r.HandleFunc("/api/programs", getPrograms).Methods("GET")
	r.HandleFunc("/api/check", checkProgramsHandler).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/recommend", recommendProgramsHandler).Methods("POST", "OPTIONS")

	// 3. 啟動伺服器：務必監聽 "0.0.0.0"
	fmt.Printf("伺服器已啟動於 Port %s...\n", port)
	err = http.ListenAndServe(":"+port, commonMiddleware(r))
	if err != nil {
		fmt.Printf("伺服器啟動失敗: %v\n", err)
	}
}
