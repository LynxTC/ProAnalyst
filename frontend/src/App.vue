<script setup>
import { ref, onMounted, computed } from 'vue';

// --- 狀態管理 ---
const BACKEND_URL = import.meta.env.VITE_API_BASE_URL || ''; // Go 後端服務地址

const programs = ref({}); // 所有學程定義 {id: {name, min_credits, ...}}
const selectedProgramIds = ref([]); // 選取的學程 ID 列表
const studentFile = ref(null); // 上傳的 JSON 檔案
const uploadStatus = ref(''); // 檔案上傳狀態訊息
const programSelectionStatus = ref(''); // 學程選擇狀態訊息
const checkResults = ref([]); // 檢核結果列表
const isChecking = ref(false); // 檢核按鈕 loading 狀態

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
        programs.value = await response.json();
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
const startCheck = async () => {
    programSelectionStatus.value = '';
    
    if (!studentFile.value) {
        uploadStatus.value = '請先上傳全人資料 JSON 檔案。';
        return;
    }

    if (selectedProgramIds.value.length === 0) {
        programSelectionStatus.value = '請至少選取一個學分學程。';
        return;
    }

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

const isReadyToCheck = computed(() => {
    return studentFile.value !== null && selectedProgramIds.value.length > 0 && !isChecking.value;
});

const buttonText = computed(() => {
    if (isChecking.value) return '檢核中...';
    if (!studentFile.value) return '請先上傳檔案';
    if (selectedProgramIds.value.length === 0) return '請選取學程後點擊開始檢核';
    return '開始檢核';
});

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
        <h1 class="text-3xl font-extrabold text-blue-800 mb-2">國立政治大學 學分學程修習檢核</h1>
        <p class="text-gray-600 mb-6 border-b pb-4">上傳全人資料，選取欲檢核的學分學程/微學程，即可查看修習進度。</p>

        <div class="mb-8 p-4 border border-blue-200 bg-blue-50 rounded-lg">
            <h2 class="text-xl font-semibold text-blue-700 mb-3 flex items-center">
                <span class="inline-flex items-center justify-center w-8 h-8 mr-3 bg-blue-500 text-white text-lg font-bold rounded-full">1</span>
                上傳全人資料 (JSON 檔)
            </h2>
            <input type="file" id="jsonFile" accept=".json" @change="handleFileChange" class="w-full text-sm text-gray-500 file:mr-4 file:py-2 file:px-4 file:rounded-full file:border-0 file:text-sm file:font-semibold file:bg-blue-100 file:text-blue-700 hover:file:bg-blue-200 transition duration-150">
            <p id="uploadStatus" class="mt-2 text-sm" :class="{
                'text-emerald-600': uploadStatus.includes('檔案已載入') || uploadStatus.includes('檢核完成'),
                'text-red-600': uploadStatus.includes('錯誤'),
                'text-gray-500': uploadStatus.includes('請先上傳')
            }">{{ uploadStatus }}</p>
        </div>

        <div class="mb-8 p-4 border border-green-200 bg-green-50 rounded-lg">
            <h2 class="text-xl font-semibold text-green-700 mb-4 flex items-center">
                <span class="inline-flex items-center justify-center w-8 h-8 mr-3 bg-green-500 text-white text-lg font-bold rounded-full">2</span>
                選取欲檢核的學分學程 (可複選)
            </h2>
            <div id="programCheckboxes" class="grid grid-cols-1 sm:grid-cols-2 gap-4">
                <div v-for="(program, id) in programs" :key="id" class="flex items-start">
                    <input :id="id" type="checkbox" :value="id" v-model="selectedProgramIds" class="h-5 w-5 text-indigo-600 border-gray-300 rounded focus:ring-indigo-500">
                    <label :for="id" class="ml-3 text-sm font-medium text-gray-700">
                        {{ program.name }}
                        <p class="text-xs text-gray-500 mt-0.5">{{ program.description.split('，')[0] }}</p>
                    </label>
                </div>
                <div v-if="Object.keys(programs).length === 0" class="text-sm text-red-500">
                    載入學程清單中...
                </div>
            </div>
            <p id="programSelectionStatus" class="mt-4 text-sm text-red-500" v-show="programSelectionStatus">{{ programSelectionStatus }}</p>
        </div>

        <div class="mb-8">
            <button 
                id="checkButton" 
                @click="startCheck"
                :disabled="!isReadyToCheck || isChecking"
                class="w-full py-3 px-6 bg-indigo-600 hover:bg-indigo-700 text-white font-bold rounded-lg shadow-lg transition duration-200 disabled:opacity-50 disabled:cursor-not-allowed">
                <span id="buttonText">{{ buttonText }}</span>
            </button>
        </div>

        <div class="mt-10 pt-6 border-t border-gray-200">
            <h2 class="text-2xl font-bold text-gray-800 mb-4">檢核結果</h2>
            <div id="resultsArea" class="space-y-6">
                <p v-if="checkResults.length === 0 && !isChecking" class="text-gray-500">檢核結果將顯示在此處。</p>

                <div v-for="result in checkResults" :key="result.programName" class="border-2 p-5 rounded-xl shadow-md" :class="result.isCompleted ? 'bg-emerald-100 border-emerald-500 text-emerald-800' : 'bg-rose-100 border-rose-500 text-rose-800'">
                    <div class="flex items-center justify-between mb-4 pb-2 border-b border-gray-300">
                        <h3 class="text-xl font-bold">{{ result.programName }}</h3>
                        <span class="px-3 py-1 text-lg font-extrabold rounded-full" :class="result.isCompleted ? 'bg-emerald-500 text-white' : 'bg-rose-500 text-white'">
                            {{ result.isCompleted ? '✓ 已修畢' : '✗ 未修畢' }}
                        </span>
                    </div>

                    <p class="text-gray-700 mb-4">{{ result.programDescription }}</p>

                    <div class="mb-4 p-3 bg-white rounded-lg border border-gray-200">
                        <h4 class="text-md font-semibold text-gray-800 mb-2">學分總計檢核</h4>
                        <div class="flex justify-between text-sm">
                            <span class="font-medium">應修總學分:</span>
                            <span :class="result.totalCreditsMet ? 'text-emerald-600 font-bold' : 'text-rose-600'">{{ result.minRequiredCredits }} 學分</span>
                        </div>
                        <div class="flex justify-between text-sm">
                            <span class="font-medium">已通過學分:</span>
                            <span :class="result.totalCreditsMet ? 'text-emerald-600 font-bold' : 'text-rose-600'">{{ result.totalPassedCredits }} 學分</span>
                        </div>
                        <p class="mt-2 text-xs" :class="result.totalCreditsMet ? 'text-emerald-600' : 'text-rose-600'">
                            {{ result.totalCreditsMet ? '總學分要求已達成。' : '總學分要求尚未達成。' }}
                        </p>
                    </div>
                    
                    <h4 class="text-lg font-semibold text-gray-800 mb-2">課程分類要求檢核</h4>
                    
                    <div v-for="cat in result.categoryResults" :key="cat.category" class="mb-3 p-3 rounded-lg border border-gray-200" :class="cat.isMet ? 'text-emerald-600 bg-emerald-50' : 'text-rose-600 bg-rose-50'">
                        <div class="flex justify-between items-center text-sm font-medium">
                            <span>{{ cat.category }}</span>
                            <span>
                                <template v-if="cat.requiredCount > 0">
                                    已修畢: <span class="font-bold">{{ cat.passedCount }} 門</span> / 應修: <span class="font-bold">{{ cat.requiredCount }} 門</span>
                                </template>
                                <template v-else>
                                    門數無強制要求 (依總學分認定)
                                </template>
                            </span>
                        </div>
                        <p class="text-xs mt-1">狀態: <span class="font-semibold">{{ cat.isMet ? '已達成' : '未達成' }}</span></p>
                        <div class="mt-2 text-xs text-gray-700">
                            <p class="font-semibold mb-1">已通過課程 ({{ cat.passedCourses.length }} 筆紀錄):</p>
                            <ul class="list-disc list-inside ml-2 max-h-32 overflow-y-auto custom-scrollbar bg-white p-2 rounded">
                                <li v-if="cat.passedCourses.length === 0">無符合要求的已通過課程。</li>
                                <li v-for="c in cat.passedCourses" :key="c.name + c.semester">{{ c.name }} ({{ c.credit.toFixed(1) }} 學分, {{ c.score }} 分)</li>
                            </ul>
                        </div>
                    </div>
                    
                    <div 
                        v-if="result.inProgressCourses && result.inProgressCourses.length > 0" 
                        class="mt-4 p-3 border border-yellow-400 bg-yellow-50 rounded-lg">
                        
                        <h4 class="text-lg font-semibold text-yellow-800 mb-2">
                            修習中課程 ({{ result.inProgressCourses ? result.inProgressCourses.length : 0 }} 門)
                        </h4>
                        
                        <p class="text-sm text-yellow-700 mb-2">以下課程成績尚未送達，若及格可能影響學程完成狀態：</p>
                        <ul class="list-disc list-inside ml-2 text-sm text-yellow-900 max-h-32 overflow-y-auto custom-scrollbar bg-white p-2 rounded">
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