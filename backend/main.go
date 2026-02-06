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
}

// 學程要求中的一個分類
type ProgramRequirement struct {
	Category   string   `json:"category"`
	MinCount   int      `json:"min_count"`
	MaxCount   int      `json:"max_count"`
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
	IsMet           bool            `json:"isMet"`
	PassedCourses   []StudentCourse `json:"passedCourses"`
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
	// ... 其他不需要的欄位
}

// 頂層結構：用於匹配 JSON 檔案外層的陣列
type StudentDataWrapper []struct {
	AcademicInfo AcademicInfo `json:"課業學習"`
}

// --- 全局變數 ---
var programs map[string]Program
var programsByCollege map[string]map[string]Program

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
		"micro_programs.json":  "micro",
		"credit_programs.json": "credit",
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
	return nil
}

// 解析並扁平化學生的歷年成績資料。
func loadStudentData(data []byte) ([]StudentCourse, error) {
	var rawData StudentDataWrapper

	if err := json.Unmarshal(data, &rawData); err != nil {
		return nil, fmt.Errorf("解析頂層 JSON 結構失敗: %w", err)
	}

	if len(rawData) == 0 || len(rawData[0].AcademicInfo.GradeRecordList) == 0 {
		return nil, fmt.Errorf("JSON 結構不符預期或未找到課程紀錄")
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
		return nil, fmt.Errorf("檔案解析成功，但未找到有效的課程紀錄")
	}

	return flatCourses, nil
}

// 核心檢核邏輯 (與原 JS checkProgramCompletion 邏輯對應)
// 檢核學生課程是否符合指定學分學程的要求。
// 注意：本函式依賴於全局變數 `programs`
func checkProgramCompletion(programID string, courses []StudentCourse) CheckResult {
	program, ok := programs[programID]
	if !ok {
		// 如果學程 ID 無效，回傳一個錯誤結果
		return CheckResult{ProgramName: fmt.Sprintf("學程 ID %s 不存在", programID)}
	}

	// var completedCourses []StudentCourse  // 儲存所有已通過的學程相關課程 (已移至下方定義)
	var inProgressCourses []StudentCourse // 儲存所有修習中的學程相關課程

	// 找出所有學程要求中涉及的課程名稱集合 (使用清理後的名稱作為 Key)
	programCourseNamesClean := make(map[string]bool)
	geCourseNames := make(map[string]bool) // 通識課程集合

	// 建立 Requirements 的副本，以便進行特殊處理 (如移除課程名稱中的教師名) 而不影響全域變數
	localRequirements := make([]ProgramRequirement, len(program.Requirements))
	for i, req := range program.Requirements {
		localRequirements[i] = req
		coursesCopy := make([]string, len(req.Courses))
		copy(coursesCopy, req.Courses)
		localRequirements[i].Courses = coursesCopy
	}

	// 特殊處理：東南亞區域研究微學程 (southeast_asian_area_studies)
	// 建立 課程名稱 -> 教師名稱 的對應表
	courseInstructorMap := make(map[string]string)
	isSoutheastAsianProgram := programID == "southeast_asian_area_studies"

	for i, req := range localRequirements {
		for j, courseName := range req.Courses {
			// 處理 "課程名稱(教師名)" 的格式，僅針對特定學程
			if isSoutheastAsianProgram && strings.Contains(courseName, "(") && strings.HasSuffix(courseName, ")") {
				start := strings.LastIndex(courseName, "(")
				realName := strings.TrimSpace(courseName[:start])
				instructor := courseName[start+1 : len(courseName)-1]

				// 更新副本中的課程名稱為真實名稱，以便後續比對
				localRequirements[i].Courses[j] = realName

				// 記錄該課程對應的教師
				norm := normalizeCourseName(realName)
				courseInstructorMap[norm] = instructor

				// 使用真實名稱加入檢查清單
				programCourseNamesClean[norm] = true
			} else {
				// 對 programs.json 中的名稱進行標準化處理
				programCourseNamesClean[normalizeCourseName(courseName)] = true
			}
		}
	}
	// 加入通識課程定義
	for _, courseName := range program.GeneralEducationCourses {
		norm := normalizeCourseName(courseName)
		programCourseNamesClean[norm] = true
		geCourseNames[norm] = true
	}

	// 步驟 1: 篩選與學程相關的課程，並處理通識課程限修一門的規則
	var relevantPassed []StudentCourse
	for _, course := range courses {
		if _, ok := programCourseNamesClean[course.Name]; ok {
			if course.IsPassed {
				relevantPassed = append(relevantPassed, course)
			} else if course.IsInProgress {
				inProgressCourses = append(inProgressCourses, course)
			}
		}
	}

	// 特殊處理：東南亞區域研究微學程 - 同一名老師開設課程至多認列兩門
	if isSoutheastAsianProgram {
		instructorCounts := make(map[string]int)
		var filteredByInstructor []StudentCourse

		// 先對已通過課程排序 (學分高者優先)，確保剔除的是學分較少或較不重要的
		sort.Slice(relevantPassed, func(i, j int) bool {
			return relevantPassed[i].Credit > relevantPassed[j].Credit
		})

		for _, c := range relevantPassed {
			norm := normalizeCourseName(c.Name)
			if instructor, ok := courseInstructorMap[norm]; ok {
				if instructorCounts[instructor] < 2 {
					instructorCounts[instructor]++
					filteredByInstructor = append(filteredByInstructor, c)
				}
			} else {
				filteredByInstructor = append(filteredByInstructor, c)
			}
		}
		relevantPassed = filteredByInstructor
	}

	// 處理通識課程：若超過一門，僅保留學分最高者
	var gePassed []StudentCourse
	var otherPassed []StudentCourse
	for _, c := range relevantPassed {
		if geCourseNames[c.Name] {
			gePassed = append(gePassed, c)
		} else {
			otherPassed = append(otherPassed, c)
		}
	}
	if len(gePassed) > 1 {
		sort.Slice(gePassed, func(i, j int) bool {
			return gePassed[i].Credit > gePassed[j].Credit
		})
		gePassed = gePassed[:1]
	}
	completedCourses := append(otherPassed, gePassed...)

	totalPassedCredits := 0.0 // 將由後續計算有效學分決定

	// 步驟 2 & 3: 檢核分類要求 (門數) 和總學分
	var categoryResults []CategoryResult
	allCategoriesMet := true

	// 特殊處理：管理會計專業學程 (management_accounting)
	isManagementAccounting := programID == "management_accounting"

	if isManagementAccounting {
		// 規則：若「經濟學」修習未達 6 學分，則不採計（以 0 學分計）
		econCredits := 0.0
		totalPassedCredits = 0.0
		for _, c := range completedCourses {
			totalPassedCredits += c.Credit
			if c.Name == "經濟學" {
				econCredits += c.Credit
			}
		}

		if econCredits < 6.0 {
			totalPassedCredits -= econCredits
			// 從已通過課程列表中移除經濟學，以免誤導
			var newCompleted []StudentCourse
			for _, c := range completedCourses {
				if c.Name != "經濟學" {
					newCompleted = append(newCompleted, c)
				}
			}
			completedCourses = newCompleted
		}

		isMet := totalPassedCredits >= program.MinCredits

		// 計算不重複的已通過課程門數
		uniquePassedCourseNames := make(map[string]bool)
		for _, c := range completedCourses {
			uniquePassedCourseNames[c.Name] = true
		}
		passedCount := len(uniquePassedCourseNames)

		categoryResults = append(categoryResults, CategoryResult{
			Category:      program.Requirements[0].Category,
			RequiredCount: 0, // 門數非強制要求
			PassedCount:   passedCount,
			IsMet:         isMet,
			PassedCourses: completedCourses, // 直接將所有已通過課程賦予
		})
		allCategoriesMet = isMet
	} else {
		effectiveTotalCredits := 0.0 // 用於計算考慮 MaxCount 後的有效總學分

		// 一般學程邏輯
		for _, req := range localRequirements {
			var passedInThisCategory []StudentCourse // 該分類下已通過的課程紀錄

			// 找出該類別已通過課程
			for _, c := range completedCourses {
				// 檢查該課程是否在當前分類的要求列表中
				isMatch := false
				for _, reqCourseName := range req.Courses {
					if c.Name == reqCourseName {
						isMatch = true
						break
					}
				}
				if isMatch {
					passedInThisCategory = append(passedInThisCategory, c)
				}
			}

			// 根據學分由高到低排序，以便在有 MaxCount 限制時優先採計高學分課程
			sort.Slice(passedInThisCategory, func(i, j int) bool {
				return passedInThisCategory[i].Credit > passedInThisCategory[j].Credit
			})

			// 計算不重複的已通過課程門數
			uniquePassedCourseNames := make(map[string]bool)
			passedCreditsInCategory := 0.0
			for _, c := range passedInThisCategory {
				uniquePassedCourseNames[c.Name] = true
				passedCreditsInCategory += c.Credit
			}
			passedCount := len(uniquePassedCourseNames)

			// 計算該分類貢獻的有效學分 (處理 MaxCount)
			creditsContributingToTotal := 0.0
			limit := len(passedInThisCategory)
			if req.MaxCount > 0 && limit > req.MaxCount {
				limit = req.MaxCount
			}
			for i := 0; i < limit; i++ {
				creditsContributingToTotal += passedInThisCategory[i].Credit
			}
			if !strings.HasPrefix(req.Category, "先修課程") {
				effectiveTotalCredits += creditsContributingToTotal
			}

			// 判斷該分類是否符合要求 (MinCount, MinCredits)
			isMet := passedCount >= req.MinCount
			if req.MinCredits > 0 && passedCreditsInCategory < req.MinCredits {
				isMet = false
			}
			if !isMet {
				allCategoriesMet = false
			}

			// 修正：確保 passedInThisCategory 在沒有課程時是空切片
			if passedInThisCategory == nil {
				passedInThisCategory = []StudentCourse{}
			}

			categoryResults = append(categoryResults, CategoryResult{
				Category:        req.Category,
				RequiredCount:   req.MinCount,
				RequiredCredits: req.MinCredits,
				PassedCount:     passedCount,
				IsMet:           isMet,
				PassedCourses:   passedInThisCategory,
			})
		}

		// 更新總通過學分為計算後的有效學分
		totalPassedCredits = effectiveTotalCredits
	}

	// 步驟 4: 總結
	totalCreditsMet := totalPassedCredits >= program.MinCredits
	isCompleted := totalCreditsMet && allCategoriesMet

	// 修正：確保 inProgressCourses 在沒有課程時是空切片 (為了解決前端 'null' 錯誤)
	if inProgressCourses == nil {
		inProgressCourses = []StudentCourse{}
	}
	// 確保 CategoryResults 內部的 PassedCourses 不為 nil
	for i := range categoryResults {
		// Go 語言中，切片在未賦值的情況下可能為 nil，序列化後即為 null。
		if categoryResults[i].PassedCourses == nil {
			categoryResults[i].PassedCourses = []StudentCourse{}
		}
	}
	// 修正：確保 categoryResults 在沒有課程時是空切片 (如果結構允許)
	if categoryResults == nil {
		categoryResults = []CategoryResult{}
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

// 處理檔案上傳和檢核
func checkProgramsHandler(w http.ResponseWriter, r *http.Request) {
	// 處理 OPTIONS 請求 (CORS 預檢)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// 1. 解析 multipart 表單
	err := r.ParseMultipartForm(32 << 20) // 32MB
	if err != nil {
		http.Error(w, "解析表單失敗: "+err.Error(), http.StatusBadRequest)
		return
	}

	// 2. 獲取選取的學程 ID
	programIDsStr := r.PostFormValue("program_ids")
	if programIDsStr == "" {
		http.Error(w, "請選取至少一個學程 ID", http.StatusBadRequest)
		return
	}
	// 假設前端傳送的是逗號分隔的 ID 字串
	programIDs := strings.Split(programIDsStr, ",")

	// 3. 讀取學生 JSON 檔案
	file, _, err := r.FormFile("student_json")
	if err != nil {
		http.Error(w, "讀取檔案失敗: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "讀取檔案內容失敗: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 4. 解析學生課程資料
	studentCourses, err := loadStudentData(fileBytes)
	if err != nil {
		http.Error(w, "解析學生資料失敗: "+err.Error(), http.StatusBadRequest)
		return
	}

	// 5. 執行檢核
	var results []CheckResult
	for _, id := range programIDs {
		result := checkProgramCompletion(id, studentCourses)
		results = append(results, result)
	}

	// 6. 回傳結果
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
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

	r := mux.NewRouter()
	// ... 你的路由設定 ...

	r.HandleFunc("/healthcheck", healthCheckHandler).Methods("GET")
	r.HandleFunc("/api/programs", getPrograms).Methods("GET")
	r.HandleFunc("/api/check", checkProgramsHandler).Methods("POST", "OPTIONS")

	// 3. 啟動伺服器：務必監聽 "0.0.0.0"
	fmt.Printf("伺服器已啟動於 Port %s...\n", port)
	err = http.ListenAndServe(":"+port, commonMiddleware(r))
	if err != nil {
		fmt.Printf("伺服器啟動失敗: %v\n", err)
	}
}
