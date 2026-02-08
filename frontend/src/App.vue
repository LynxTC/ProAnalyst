<script setup>
import { ref, onMounted, computed } from 'vue';

// --- 狀態管理 ---
const BACKEND_URL = import.meta.env.VITE_API_BASE_URL || ''; // Go 後端服務地址

// 定義學院顯示順序
const COLLEGE_ORDER = [
    "文學院", "社會科學學院", "商學院", "傳播學院", "外國語文學院",
    "法學院", "理學院", "國際事務學院", "教育學院", "創新國際學院",
    "資訊學院", "X實驗學院"
];

const programsByCollege = ref({}); // 所有學程定義 (按學院分類)
const selectedCollege = ref(''); // 目前選擇的學院
const selectedProgramType = ref('credit'); // 目前選擇的學程類型 ('credit' | 'micro')
const selectedProgramIds = ref([]); // 選取的學程 ID 列表
const studentFile = ref(null); // 上傳的 JSON 檔案
const uploadStatus = ref(''); // 檔案上傳狀態訊息
const programSelectionStatus = ref(''); // 學程選擇狀態訊息
const checkResults = ref([]); // 檢核結果列表
const isChecking = ref(false); // 檢核按鈕 loading 狀態
const showDownloadHelp = ref(false); // 是否顯示下載說明
const showDisclaimerModal = ref(false); // 是否顯示免責聲明 Modal

// --- 核心邏輯 ---

/**
 * 步驟 1: 載入學程列表
 */
const loadPrograms = async () => {
    try {
        const response = await fetch(`${BACKEND_URL}/api/programs`);
        if (!response.ok) {
            throw new Error(`HTTP error! status: ${response.status}`);
        }
        programsByCollege.value = await response.json();
        // 預設選取第一個學院
        if (sortedCollegeNames.value.length > 0) {
            selectedCollege.value = sortedCollegeNames.value[0];
        }
    } catch (error) {
        console.error("載入學程列表失敗:", error);
        alert('無法連線到 Go 後端服務 (請確認 Go 程式已執行於 :8080)');
    }
};

/**
 * 步驟 2: 處理檔案選擇
 */
const handleFileChange = (event) => {
    const file = event.target.files[0];
    studentFile.value = file;
    checkResults.value = []; // 清空結果

    if (!file) {
        uploadStatus.value = '';
        return;
    }

    if (file.type !== 'application/json') {
        uploadStatus.value = '錯誤：請確保上傳的檔案是 JSON 格式 (.json)。';
        studentFile.value = null;
        return;
    }

    // 可以在這裡執行初步的檔案大小/名稱檢查
    uploadStatus.value = `檔案已載入: ${file.name} (${(file.size / 1024).toFixed(2)} KB)。`;
};

/**
 * 步驟 3: 執行檢核
 */
const startCheck = () => {
    programSelectionStatus.value = '';

    if (!studentFile.value) {
        uploadStatus.value = '請先上傳全人資料 JSON 檔案。';
        return;
    }

    if (selectedProgramIds.value.length === 0) {
        programSelectionStatus.value = '請至少選取一個學分學程。';
        return;
    }

    showDisclaimerModal.value = true;
};

const executeCheck = async () => {
    showDisclaimerModal.value = false;
    isChecking.value = true;
    checkResults.value = [];

    const formData = new FormData();
    formData.append('student_json', studentFile.value);
    formData.append('program_ids', selectedProgramIds.value.join(','));

    try {
        const response = await fetch(`${BACKEND_URL}/api/check`, {
            method: 'POST',
            body: formData,
        });

        if (!response.ok) {
            const errorText = await response.text();
            throw new Error(`檢核失敗: ${response.status} - ${errorText}`);
        }

        checkResults.value = await response.json();
        uploadStatus.value = `檢核完成。共檢核 ${checkResults.value.length} 個學程。`;

    } catch (error) {
        console.error("執行檢核失敗:", error);
        alert(`檢核過程中發生錯誤: ${error.message}`);
    } finally {
        isChecking.value = false;
    }
};

// --- Computed 屬性 (用於 UI 邏輯) ---

const sortedCollegeNames = computed(() => {
    return Object.keys(programsByCollege.value).sort((a, b) => {
        const indexA = COLLEGE_ORDER.indexOf(a);
        const indexB = COLLEGE_ORDER.indexOf(b);

        if (indexA !== -1 && indexB !== -1) return indexA - indexB;
        if (indexA !== -1) return -1;
        if (indexB !== -1) return 1;
        return a.localeCompare(b);
    });
});

const currentPrograms = computed(() => {
    if (!selectedCollege.value) return {};
    const programs = programsByCollege.value[selectedCollege.value] || {};

    // 根據選擇的類型過濾學程
    const filtered = {};
    for (const [id, p] of Object.entries(programs)) {
        if (p.type === selectedProgramType.value) {
            filtered[id] = p;
        }
    }
    return filtered;
});

const isReadyToCheck = computed(() => {
    return studentFile.value !== null && selectedProgramIds.value.length > 0 && !isChecking.value;
});

const buttonText = computed(() => {
    if (isChecking.value) return '檢核中...';
    if (!studentFile.value) return '請先上傳檔案';
    if (selectedProgramIds.value.length === 0) return '請選取學程後點擊開始檢核';
    return '開始檢核';
});

const selectedProgramsList = computed(() => {
    const list = [];
    for (const college of Object.values(programsByCollege.value)) {
        for (const [id, program] of Object.entries(college)) {
            if (selectedProgramIds.value.includes(id)) {
                list.push({ id, name: program.name });
            }
        }
    }
    return list;
});

const removeProgram = (id) => {
    selectedProgramIds.value = selectedProgramIds.value.filter(pid => pid !== id);
};

// --- Lifecycle 鉤子 ---
onMounted(() => {
    loadPrograms();
});

// 假設 checkResults 是一個 ref([])

const safeCheckResults = computed(() => {
    return checkResults.value.map(result => ({
        ...result,
        inProgressCourses: result.inProgressCourses || [], // 如果是 null，強制變為空陣列
        // 對所有陣列屬性進行同樣處理，例如：
        // categoryResults: result.categoryResults || []
    }));
});
</script>

<template>
    <div class="max-w-4xl mx-auto bg-white shadow-2xl rounded-xl p-6 sm:p-10">
        <h1 class="text-3xl font-extrabold text-blue-800 mb-2">國立政治大學 學分學程 / 微學程修習檢核</h1>
        <p class="text-gray-600 mb-6 border-b pb-4">上傳全人資料，選取欲檢核的學分學程/微學程，即可查看修習進度。</p>

        <div class="mb-8 p-4 border border-blue-200 bg-blue-50 rounded-lg">
            <h2 class="text-xl font-semibold text-blue-700 mb-3 flex items-center">
                <span
                    class="inline-flex items-center justify-center w-8 h-8 mr-3 bg-blue-500 text-white text-lg font-bold rounded-full">1</span>
                上傳全人資料 (JSON 檔)
                <span @click="showDownloadHelp = !showDownloadHelp"
                    class="ml-3 text-sm text-gray-400 cursor-pointer hover:text-gray-600 underline decoration-dotted transition-colors select-none">
                    如何下載全人資料?
                </span>
            </h2>
            <div v-if="showDownloadHelp"
                class="mb-4 p-4 bg-white border border-blue-100 rounded-lg shadow-sm text-sm text-gray-600 leading-relaxed">
                <p class="mb-1"><span class="font-bold">Step 1️⃣：</span>進入政大首頁並且登入 iNCCU</p>
                <p class="mb-1"><span class="font-bold">Step 2️⃣：</span>點選「進入我的全人」</p>
                <p class="mb-1"><span class="font-bold">Step 3️⃣：</span>下滑到底，在「相關連結」找到「資料格式化匯出」選項，進入後選擇「課業學習」後下載</p>
                <p><span class="font-bold">Step 4️⃣：</span>得到熱騰騰的全人資料 JSON 檔案！</p>
            </div>
            <input type="file" id="jsonFile" accept=".json" @change="handleFileChange"
                class="w-full text-sm text-gray-500 file:mr-4 file:py-2 file:px-4 file:rounded-full file:border-0 file:text-sm file:font-semibold file:bg-blue-100 file:text-blue-700 hover:file:bg-blue-200 transition duration-150">
            <p id="uploadStatus" class="mt-2 text-sm" :class="{
                'text-emerald-600': uploadStatus.includes('檔案已載入') || uploadStatus.includes('檢核完成'),
                'text-red-600': uploadStatus.includes('錯誤'),
                'text-gray-500': uploadStatus.includes('請先上傳')
            }">{{ uploadStatus }}</p>
        </div>

        <div class="mb-8 p-4 border border-green-200 bg-green-50 rounded-lg">
            <h2 class="text-xl font-semibold text-green-700 mb-4 flex items-center">
                <span
                    class="inline-flex items-center justify-center w-8 h-8 mr-3 bg-green-500 text-white text-lg font-bold rounded-full">2</span>
                選取欲檢核的學分學程 (可複選)
            </h2>

            <div class="flex flex-col sm:flex-row sm:items-end gap-4 mb-6">
                <!-- 學院選擇下拉選單 -->
                <div class="w-full sm:w-1/2">
                    <label for="collegeSelect" class="block text-sm font-medium text-gray-700 mb-1">選擇設置單位或所屬學院：</label>
                    <select id="collegeSelect" v-model="selectedCollege"
                        class="block w-full pl-3 pr-10 py-2 text-base border border-gray-300 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 sm:text-sm rounded-md">
                        <option v-for="collegeName in sortedCollegeNames" :key="collegeName" :value="collegeName">{{
                            collegeName }}</option>
                    </select>
                </div>

                <!-- 學程類型選擇 (Radio Buttons) -->
                <div class="flex items-center space-x-6 pb-2">
                    <label class="inline-flex items-center cursor-pointer">
                        <input type="radio" value="credit" v-model="selectedProgramType"
                            class="h-4 w-4 text-indigo-600 border-gray-300 focus:ring-indigo-500">
                        <span class="ml-2 text-gray-700 font-medium">學分學程</span>
                    </label>
                    <label class="inline-flex items-center cursor-pointer">
                        <input type="radio" value="micro" v-model="selectedProgramType"
                            class="h-4 w-4 text-indigo-600 border-gray-300 focus:ring-indigo-500">
                        <span class="ml-2 text-gray-700 font-medium">微學程</span>
                    </label>
                </div>
            </div>

            <p v-if="selectedProgramType === 'credit'" class="text-sm text-gray-500 mb-4">
                註：學分學程認列科目至少應有三分之一學分數不屬於原學系、所之專業必修科目（此檢核項目尚未建置，請使用者自行確認）
            </p>

            <p v-if="selectedProgramType === 'micro'" class="text-sm text-gray-500 mb-4">
                註：微學程所認列之通識課程以一門為限（以學分較多者計）
            </p>

            <div id="programCheckboxes" class="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <div v-for="(program, id) in currentPrograms" :key="id" class="flex items-start">
                    <input :id="id" type="checkbox" :value="id" v-model="selectedProgramIds"
                        class="h-5 w-5 text-indigo-600 border-gray-300 rounded focus:ring-indigo-500 shrink-0">
                    <label :for="id" class="ml-3 text-sm font-medium text-gray-700">
                        {{ program.name }}
                        <p class="text-xs text-gray-500 mt-0.5">{{ program.description }}</p>
                    </label>
                </div>
                <div v-if="Object.keys(programsByCollege).length === 0" class="text-sm text-red-500">
                    載入學程清單中...
                </div>
            </div>
            <p id="programSelectionStatus" class="mt-4 text-sm text-red-500" v-show="programSelectionStatus">{{
                programSelectionStatus }}</p>

            <!-- 顯示已選擇的學程 -->
            <div v-if="selectedProgramsList.length > 0" class="mt-4 pt-4 border-t border-green-200">
                <p class="text-sm font-bold text-green-800 mb-2">已選擇的學程（點擊可取消）：</p>
                <div class="flex flex-wrap gap-2">
                    <span v-for="p in selectedProgramsList" :key="p.id" @click="removeProgram(p.id)"
                        class="px-3 py-1 bg-white text-green-700 text-sm font-medium rounded-full border border-green-300 shadow-sm cursor-pointer hover:bg-red-50 hover:text-red-600 hover:border-red-300 transition-colors flex items-center group">
                        {{ p.name }}
                    </span>
                </div>
            </div>
        </div>

        <div class="mb-8">
            <button id="checkButton" @click="startCheck" :disabled="!isReadyToCheck || isChecking"
                class="w-full py-3 px-6 bg-indigo-600 hover:bg-indigo-700 text-white font-bold rounded-lg shadow-lg transition duration-200 disabled:opacity-50 disabled:cursor-not-allowed">
                <span id="buttonText">{{ buttonText }}</span>
            </button>
        </div>

        <div class="mt-10 pt-6 border-t border-gray-200">
            <h2 class="text-2xl font-bold text-gray-800 mb-4">檢核結果</h2>
            <div id="resultsArea" class="space-y-6">
                <p v-if="checkResults.length === 0 && !isChecking" class="text-gray-500">檢核結果將顯示在此處。</p>

                <div v-for="result in checkResults" :key="result.programName" class="border-2 p-5 rounded-xl shadow-md"
                    :class="result.isCompleted ? 'bg-emerald-100 border-emerald-500 text-emerald-800' : 'bg-rose-100 border-rose-500 text-rose-800'">
                    <div class="flex items-center justify-between mb-4 pb-2 border-b border-gray-300">
                        <h3 class="text-xl font-bold">{{ result.programName }}</h3>
                        <span class="px-3 py-1 text-lg font-extrabold rounded-full"
                            :class="result.isCompleted ? 'bg-emerald-500 text-white' : 'bg-rose-500 text-white'">
                            {{ result.isCompleted ? '✓ 已修畢' : '✗ 未修畢' }}
                        </span>
                    </div>

                    <p class="text-gray-700 mb-4">{{ result.programDescription }}</p>

                    <div class="mb-4 p-3 bg-white rounded-lg border border-gray-200">
                        <h4 class="text-md font-semibold text-gray-800 mb-2">學分總計檢核</h4>
                        <div class="flex justify-between text-sm">
                            <span class="font-medium">應修總學分:</span>
                            <span :class="result.totalCreditsMet ? 'text-emerald-600 font-bold' : 'text-rose-600'">{{
                                result.minRequiredCredits }} 學分</span>
                        </div>
                        <div class="flex justify-between text-sm">
                            <span class="font-medium">已通過學分:</span>
                            <span :class="result.totalCreditsMet ? 'text-emerald-600 font-bold' : 'text-rose-600'">{{
                                result.totalPassedCredits }} 學分</span>
                        </div>
                        <p class="mt-2 text-xs" :class="result.totalCreditsMet ? 'text-emerald-600' : 'text-rose-600'">
                            {{ result.totalCreditsMet ? '總學分要求已達成。' : '總學分要求尚未達成。' }}
                        </p>
                    </div>

                    <h4 class="text-lg font-semibold text-gray-800 mb-2">課程分類要求檢核</h4>

                    <div v-for="cat in result.categoryResults" :key="cat.category"
                        class="mb-3 p-3 rounded-lg border border-gray-200"
                        :class="((cat.requiredCount > 0 || cat.requiredCredits > 0) ? cat.isMet : result.isCompleted) ? 'text-emerald-600 bg-emerald-50' : 'text-rose-600 bg-rose-50'">
                        <div class="flex justify-between items-center text-sm font-medium">
                            <span>{{ cat.category }}</span>
                            <div class="text-right">
                                <div v-if="cat.requiredCount > 0">
                                    已修畢: <span class="font-bold">{{ cat.passedCount }} {{ cat.category.includes('跨群選修要求') ? '群' : '門' }}</span> / 應修: <span
                                        class="font-bold">{{ cat.requiredCount }} {{ cat.category.includes('跨群選修要求') ? '群' : '門' }}</span>
                                </div>
                                <div v-if="cat.requiredCredits > 0">
                                    已修畢: <span class="font-bold">{{ cat.passedCredits.toFixed(1) }} 學分</span> / 應修: <span
                                        class="font-bold">{{ cat.requiredCredits.toFixed(1) }} 學分</span>
                                </div>
                                <div v-if="cat.requiredCount === 0 && cat.requiredCredits === 0">
                                    門數/學分無強制要求 (依總學分認定)
                                </div>
                            </div>
                        </div>
                        <div v-if="cat.limitExceeded" class="text-xs font-bold text-amber-600 mt-1 flex items-center">
                            <span class="mr-1">⚠️</span>
                            {{ cat.exceededMessage }}
                        </div>
                        <p v-if="cat.requiredCount > 0 || cat.requiredCredits > 0" class="text-xs mt-1">狀態: <span
                                class="font-semibold">{{
                                cat.isMet ? '已達成' : '未達成' }}</span>
                        </p>
                        <div v-if="cat.category !== '群A + 群B 總修習門數' && cat.category !== '跨群選修要求 (A-D群至少兩群)'" class="mt-2 text-xs text-gray-700">
                            <p class="font-semibold mb-1">已通過課程 ({{ cat.passedCourses.length }} 筆紀錄):</p>
                            <ul
                                class="list-disc list-inside ml-2 max-h-32 overflow-y-auto custom-scrollbar bg-white p-2 rounded">
                                <li v-if="cat.passedCourses.length === 0">無符合要求的已通過課程。</li>
                                <li v-for="c in cat.passedCourses" :key="c.name + c.semester">{{ c.name }} ({{
                                    c.credit.toFixed(1) }} 學分<span v-if="c.isCapped"
                                        class="text-amber-600 font-bold ml-1" title="此課程因超過上限而被調整學分">*</span>, {{
                                            c.score }} 分)</li>
                            </ul>
                        </div>
                    </div>

                    <div v-if="result.inProgressCourses && result.inProgressCourses.length > 0"
                        class="mt-4 p-3 border border-yellow-400 bg-yellow-50 rounded-lg">

                        <h4 class="text-lg font-semibold text-yellow-800 mb-2">
                            修習中課程 ({{ result.inProgressCourses ? result.inProgressCourses.length : 0 }} 門)
                        </h4>

                        <p class="text-sm text-yellow-700 mb-2">以下課程成績尚未送達，若及格可能影響學程完成狀態：</p>
                        <ul
                            class="list-disc list-inside ml-2 text-sm text-yellow-900 max-h-32 overflow-y-auto custom-scrollbar bg-white p-2 rounded">
                            <li v-for="c in result.inProgressCourses" :key="c.name + c.semester">
                                {{ c.name }} ({{ c.credit.toFixed(1) }} 學分) - {{ c.semester }}
                            </li>
                        </ul>
                    </div>

                </div>
                <div v-for="result in safeCheckResults" :key="result.programName">
                </div>
            </div>
        </div>

        <!-- 免責聲明 Modal -->
        <div v-if="showDisclaimerModal"
            class="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50 backdrop-blur-sm p-4">
            <div
                class="bg-white rounded-xl shadow-2xl max-w-md w-full p-6 transform transition-all scale-100 text-center">
                <h3 class="text-xl font-bold text-gray-800 mb-4 flex items-center justify-center">
                    <span class="text-yellow-500 mr-2">⚠️</span> 注意事項
                </h3>
                <p class="text-gray-600 mb-6 leading-relaxed">
                    本系統檢核結果僅供參考<br>可能因申請年度不同或修習同名課程產生檢核誤差<br><br>
                    <span class="font-bold text-gray-800">實際修習狀態以學程設置單位認定為準</span>
                </p>
                <div class="flex justify-center space-x-3">
                    <button @click="showDisclaimerModal = false"
                        class="px-4 py-2 text-gray-600 hover:bg-gray-100 font-medium rounded-lg transition-colors">
                        取消
                    </button>
                    <button @click="executeCheck"
                        class="px-4 py-2 bg-indigo-600 hover:bg-indigo-700 text-white font-bold rounded-lg shadow-md transition-colors">
                        確定並開始檢核
                    </button>
                </div>
            </div>
        </div>
    </div>
</template>

<style>
/* 這裡是 Tailwind CSS 的自定義滾動條樣式，通常會放在 index.css 或其他全域 CSS 文件中 */
.custom-scrollbar::-webkit-scrollbar {
    width: 8px;
}

.custom-scrollbar::-webkit-scrollbar-thumb {
    background-color: #cbd5e1;
    border-radius: 4px;
}

.custom-scrollbar::-webkit-scrollbar-track {
    background: #f1f5f9;
}

body {
    font-family: 'Inter', 'Noto Sans TC', sans-serif;
    background-color: #f8fafc;
}
</style>