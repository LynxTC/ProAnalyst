# **📚 國立政治大學學程修習推薦暨檢核工具**

這是一個專為國立政治大學學生設計的開源工具，透過上傳從 iNCCU 全人系統匯出的「課業學習」JSON 檔案，即可快速檢核學分學程或微學程的修習進度，並提供智慧推薦功能，找出最適合您的學程。

## **🌐 線上版本**

本專案已正式佈署，您可以直接訪問以下網址使用：

* **前端介面：** [https://program-checker.onrender.com](https://program-checker.onrender.com)
* **後端 API 服務：** `https://program-checker-backend.onrender.com`

## **✨ 專案特色**

* **🔮 智慧學程推薦：** 系統自動分析您的修課紀錄，計算所有學程的完成度，並推薦匹配度最高的學程，讓您不錯過任何取得證書的機會。
* **✏️ 自選學程檢核：** 支援學分學程、微學程及院級專長學程（如商學院）的進度檢核，清楚列出已修畢、修習中及缺漏的科目。
* **📊 視覺化報告：** 清晰呈現各分類（如必修、選修、群組要求）的門數與學分達成率，並自動處理複雜的規則（如：通識限修一門、同性質課程擇一認列、平均成績門檻等）。
* **🔒 資料隱私優先：** 採用「資料不落地」原則，檢核過程僅在伺服器記憶體中進行運算，絕不儲存您的成績單或個人資料。
* **📱 響應式設計：** 結合 Vue 3 與 Tailwind CSS，提供流暢的手機與電腦瀏覽體驗。

## **🛠️ tech-stack**

| 類別 | 技術名稱 | 說明 |
| :---- | :---- | :---- |
| **後端 (Backend)** | Go (Golang) | 使用 `gorilla/mux` 處理路由，實作高效的核心檢核邏輯與 RESTful API。 |
| **前端 (Frontend)** | Vue 3 (Composition API) | 使用 Vite 建置，結合 Tailwind CSS 打造現代化 UI。 |
| **資料交換** | JSON | 前後端透過 JSON 格式傳遞學程定義與檢核結果。 |
| **佈署 (Deployment)** | Render | 雲端服務佈署。 |

## **📂 專案結構**

```text
program-checker/
├── backend/                         # Go 後端核心
│   ├── main.go                          # API 服務與檢核邏輯
│   ├── special_handlers.go              # 特殊學程規則與進階檢核邏輯
│   ├── data/                            # 資料庫檔案
│   │   ├── credit_programs.json             # 學分學程資料庫
│   │   ├── micro_programs.json              # 微學程資料庫
│   │   ├── commerce_specialty_programs.json # 院級專長學程資料庫
│   │   └── departments_grouped.json         # 系所歸屬定義
│   └── ...
├── frontend/                        # Vue 3 前端介面
│   ├── .env.example                     # 環境變數範例
│   └── ...
└── README.md                        # 說明文件
```

## **快速開始 (開發環境)**

如果您想在本地環境運行或參與開發，請參考以下步驟：

### **步驟 1: 後端環境 (Go)**

1. 進入後端目錄：
   ```bash
   cd backend
   ```
2. 確保已備妥學程定義檔（位於 `data/` 目錄下）：
   * `data/micro_programs.json`
   * `data/credit_programs.json`
   * `data/commerce_specialty_programs.json`
   * `data/departments_grouped.json`
3. 啟動服務 (預設 Port 8080)：
   ```bash
   go run .
   ```

### **步驟 2: 前端環境 (Vue)**

1. 進入前端目錄：
   ```bash
   cd frontend
   ```
2. 安裝依賴：
   ```bash
   npm install
   ```
3. 設定環境變數：
   複製 `.env.example` 為 `.env.local` 並設定後端位址：
   ```bash
   VITE_API_BASE_URL=http://localhost:8080
   ```
4. 啟動開發伺服器：
   ```bash
   npm run dev
   ```

## **💡 使用指南**

1. **取得資料：** 登入政大 iNCCU -> 全人系統 -> 資料格式化匯出 -> 下載「課業學習」JSON 檔。
2. **上傳檔案：** 在本工具上傳該 JSON 檔。
3. **選擇模式：**
   * **智慧推薦：** 點擊「啟動推薦分析」，查看系統計算出的高完成度學程排行。
   * **學程檢核：** 切換至「學程檢核」頁籤，手動勾選感興趣的學程（支援跨學院搜尋）。
4. **查看結果：** 閱讀詳細的檢核報告，包含學分統計、修習中課程提示及未達標原因。

## **📝 學程定義維護**

後端 `backend/data` 資料夾中的 JSON 檔案定義了各學程的規則：

* `data/credit_programs.json`: 一般學分學程
* `data/micro_programs.json`: 微學程
* `data/commerce_specialty_programs.json`: 院級專長學程（目前僅商學院使用）
* `data/departments_grouped.json`: 系所歸屬定義（用於判斷學生學藉歸屬，檢查是否牴觸學程身分限制）

### **JSON 結構說明**

若您希望協助更新學程資料，請參考以下欄位定義：

```json
"program_id": {
    "name": "學程名稱",
    "min_credits": 20.0,  // 總學分門檻
    "description": "學程通過條件描述",
    "general_education_courses": ["通識A", "通識B"], // (選填) 指定通識課程清單
    "requirements": [
        {
            "category": "必修課程",   // (必填) 認列課程類別（如必修、基礎等）
            "min_credits": 6.0,      // (選填) 該類別學分門檻
            "min_count": 2,          // (選填) 該類別門數門檻
            "max_count": 1,          // (選填) 該類別採計上限門數
            "courses": ["課程A", "課程B"]
        }
    ]
}
```

## **🤝 貢獻指南**

歡迎提交 Pull Request 來新增或修正學程資料！如果您發現某個學程的規則有誤，或是有新的學程想要加入，請直接修改上述的 JSON 檔案並提交變更。

## **⚖️ 免責聲明**

本工具為學生開發之輔助系統，檢核邏輯雖力求準確，但仍可能因學校政策變動或特殊修課狀況而有誤差。**最終修畢資格與學分認定，悉以國立政治大學教務處及各學程設置單位之正式審核結果為準。**
