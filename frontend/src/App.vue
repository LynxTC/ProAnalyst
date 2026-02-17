<script setup>
import { ref, onMounted, onUnmounted, computed } from 'vue';
import CheckResultCard from './components/CheckResultCard.vue';
import AppModals from './components/AppModals.vue';
import FileUpload from './components/FileUpload.vue';

// --- 狀態管理 ---
const BACKEND_URL = import.meta.env.VITE_API_BASE_URL || ''; // Go 後端服務地址

// 定義設置單位顯示順序
const COLLEGE_ORDER = [
    "文學院", "社會科學學院", "商學院", "傳播學院", "外國語文學院",
    "法學院", "理學院", "國際事務學院", "教育學院", "創新國際學院",
    "資訊學院", "X實驗學院", "選舉研究中心", "創新與創造力研究中心", "文山共好USR計畫"
];

const programsByCollege = ref({}); // 所有學程定義 (按學院分類)
const selectedCollege = ref(''); // 目前選擇的學院
const searchQuery = ref(''); // 搜尋關鍵字
const selectedProgramType = ref('credit'); // 目前選擇的學程類型 ('credit' | 'micro')
const selectedProgramIds = ref([]); // 選取的學程 ID 列表
const studentFile = ref(null); // 上傳的 JSON 檔案
const uploadStatus = ref(''); // 檔案上傳狀態訊息
const programSelectionStatus = ref(''); // 學程選擇狀態訊息
const checkResults = ref([]); // 檢核結果列表
const currentPage = ref(1); // 當前頁碼
const pageSize = ref(10); // 每頁顯示數量 (降低單頁渲染數以確保流暢度)
const isChecking = ref(false); // 檢核按鈕 loading 狀態
const showDisclaimerModal = ref(false); // 是否顯示免責聲明 Modal
const showCompletionModal = ref(false); // 是否顯示達標恭喜 Modal
const completedPrograms = ref([]); // 已達標的學程名稱列表
const disclaimerAction = ref('check'); // 免責聲明 Modal 的動作 ('check' | 'recommend')
const recommendationResults = ref([]); // 推薦結果
const isRecommending = ref(false); // 推薦分析 loading 狀態
const hasRunRecommendation = ref(false); // 是否已執行過推薦
const showPrivacyModal = ref(false); // 是否顯示隱私權政策 Modal
const showTermsModal = ref(false); // 是否顯示服務條款 Modal
const showContactModal = ref(false); // 是否顯示聯絡我們 Modal
const activeTab = ref('recommendation'); // 當前顯示的頁籤 ('recommendation' | 'check')
const notification = ref({ show: false, message: '', type: 'success', action: null }); // 通知狀態
const showScrollTop = ref(false); // 是否顯示回到頂部按鈕
const showSelectedSidebar = ref(false); // 是否顯示已選學程側欄

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
const processFile = (file) => {
    studentFile.value = file;
    checkResults.value = []; // 清空結果
    recommendationResults.value = []; // 清空推薦結果
    hasRunRecommendation.value = false;

    if (!file) {
        uploadStatus.value = '';
        return;
    }

    if (file.type !== 'application/json' && !file.name.toLowerCase().endsWith('.json')) {
        uploadStatus.value = '錯誤：請確保上傳的檔案是 JSON 格式 (.json)';
        studentFile.value = null;
        return;
    }

    // 可以在這裡執行初步的檔案大小/名稱檢查
    uploadStatus.value = `檔案已載入: ${file.name} (${(file.size / 1024).toFixed(2)} KB)`;
};

/**
 * 步驟 3: 執行檢核
 */
const startCheck = () => {
    programSelectionStatus.value = '';

    if (!studentFile.value) {
        uploadStatus.value = '請先上傳全人資料 JSON 檔案';
        return;
    }

    if (selectedProgramIds.value.length === 0) {
        programSelectionStatus.value = '請至少選取一個學分學程';
        return;
    }

    disclaimerAction.value = 'check';
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
        currentPage.value = 1;
        uploadStatus.value = `檢核完成。共檢核 ${checkResults.value.length} 個學程。`;

    } catch (error) {
        console.error("執行檢核失敗:", error);
        alert(`檢核過程中發生錯誤: ${error.message}`);
    } finally {
        isChecking.value = false;
    }
};

/**
 * 額外功能: 執行學程推薦
 */
const startRecommendation = () => {
    if (!studentFile.value) {
        alert('請先上傳全人資料 JSON 檔案');
        return;
    }
    disclaimerAction.value = 'recommend';
    showDisclaimerModal.value = true;
};

const executeRecommendation = async () => {
    showDisclaimerModal.value = false;
    isRecommending.value = true;
    hasRunRecommendation.value = false;
    recommendationResults.value = [];

    const formData = new FormData();
    formData.append('student_json', studentFile.value);

    try {
        const response = await fetch(`${BACKEND_URL}/api/recommend`, {
            method: 'POST',
            body: formData,
        });

        if (!response.ok) throw new Error('推薦分析失敗');
        recommendationResults.value = await response.json();
        hasRunRecommendation.value = true;

        const completed = recommendationResults.value.filter(r => r.isCompleted);
        if (completed.length > 0) {
            completedPrograms.value = completed.map(r => r.programName);
            showCompletionModal.value = true;
        }
    } catch (error) {
        console.error("推薦分析錯誤:", error);
        alert('無法執行推薦分析');
    } finally {
        isRecommending.value = false;
    }
};

let notificationTimeout = null;
const showNotification = (message, type = 'success', action = null) => {
    notification.value = { show: true, message, type, action };
    if (notificationTimeout) clearTimeout(notificationTimeout);
    notificationTimeout = setTimeout(() => {
        notification.value.show = false;
    }, 4000);
};

const addProgramToSelection = (id, name) => {
    if (!selectedProgramIds.value.includes(id)) {
        selectedProgramIds.value.push(id);
        showNotification(`已加入「${name}」`, 'success', {
            text: '查看已選清單',
            handler: () => { showSelectedSidebar.value = true; notification.value.show = false; }
        });
    } else {
        showNotification(`「${name}」已在清單中`, 'info', {
            text: '查看已選清單',
            handler: () => { showSelectedSidebar.value = true; notification.value.show = false; }
        });
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

const getFilteredPrograms = (predicate) => {
    const query = searchQuery.value.trim().toLowerCase();
    const filtered = {};

    if (query) {
        // 搜尋模式：跨學院搜尋
        for (const collegePrograms of Object.values(programsByCollege.value)) {
            for (const [id, p] of Object.entries(collegePrograms)) {
                if (predicate(p) && p.name.toLowerCase().includes(query)) {
                    filtered[id] = p;
                }
            }
        }
    } else {
        // 一般模式：僅顯示所選學院
        if (!selectedCollege.value) return {};
        const programs = programsByCollege.value[selectedCollege.value] || {};
        for (const [id, p] of Object.entries(programs)) {
            if (predicate(p)) {
                filtered[id] = p;
            }
        }
    }
    return filtered;
};

const primaryPrograms = computed(() => {
    return getFilteredPrograms((p) => {
        return selectedProgramType.value === 'micro' ? p.type === 'micro' : p.type === 'credit';
    });
});

const secondaryPrograms = computed(() => {
    if (selectedProgramType.value !== 'credit') return {};
    return getFilteredPrograms((p) => p.type === 'specialty');
});

const isReadyToCheck = computed(() => {
    return studentFile.value !== null && selectedProgramIds.value.length > 0 && !isChecking.value;
});

const buttonText = computed(() => {
    if (isChecking.value) return '檢核中...';
    if (!studentFile.value) return '請先上傳檔案';
    if (selectedProgramIds.value.length === 0) return '請選取學程';
    return '開始確認';
});

const selectedProgramsList = computed(() => {
    const list = [];
    const seenIds = new Set();
    for (const college of Object.values(programsByCollege.value)) {
        for (const [id, program] of Object.entries(college)) {
            if (selectedProgramIds.value.includes(id) && !seenIds.has(id)) {
                list.push({ id, name: program.name });
                seenIds.add(id);
            }
        }
    }
    return list;
});

const visibleCheckResults = computed(() => {
    const start = (currentPage.value - 1) * pageSize.value;
    const end = start + pageSize.value;
    return checkResults.value.slice(start, end);
});

const rankedRecommendations = computed(() => {
    // 複製一份陣列以避免修改原始資料，並計算剩餘學分
    const list = recommendationResults.value.map(rec => {
        const totalRemaining = Math.max(0, rec.minCredits - rec.totalPassedCredits);
        let remaining = totalRemaining;

        if (rec.categoryResults && Array.isArray(rec.categoryResults)) {
            // 1. 若該學程所有分項皆設有 "min_credits" (即 requiredCredits > 0)
            const allCategoriesHaveLimit = rec.categoryResults.every(cat => (cat.requiredCredits || 0) > 0);

            if (allCategoriesHaveLimit) {
                // 將各未滿足分項剩餘學分加總
                const sumCategoryRemaining = rec.categoryResults.reduce((sum, cat) => {
                    return sum + Math.max(0, (cat.requiredCredits || 0) - cat.passedCredits);
                }, 0);
                // 並與 "總要求學分與已修學分差值" 比較，以剩餘較多者為計算標準
                remaining = Math.max(sumCategoryRemaining, totalRemaining);
            }
            // 2. 若有任一分項未制定下限，則直接以總要求學分剩餘為準 (即 remaining = totalRemaining)
        }

        return { ...rec, remaining };
    });

    if (list.length === 0) return [];

    // 排序：主要以剩餘學分 (由少至多)，次要以完成度 (由高至低)
    list.sort((a, b) => {
        if (Math.abs(a.remaining - b.remaining) > 0.1) {
            return a.remaining - b.remaining;
        }
        // 若剩餘學分相同，優先顯示已完全通過者
        if (a.isCompleted !== b.isCompleted) {
            return a.isCompleted ? -1 : 1;
        }
        return b.completionRate - a.completionRate;
    });

    // 賦予排名 (處理並列)
    let currentRank = 1;
    return list.map((rec, index) => {
        if (index > 0) {
            const prev = list[index - 1];
            const remainingDiff = Math.abs(rec.remaining - prev.remaining) > 0.1;
            const statusDiff = rec.isCompleted !== prev.isCompleted;

            if (remainingDiff || statusDiff) {
                currentRank++;
            }
        }
        return { ...rec, rank: currentRank };
    });
});

const totalPages = computed(() => {
    return Math.ceil(checkResults.value.length / pageSize.value);
});

const changePage = (page) => {
    if (page >= 1 && page <= totalPages.value) {
        currentPage.value = page;
        // 切換頁面後自動捲動到結果區頂部，避免使用者迷失方向
        const resultsArea = document.getElementById('resultsArea');
        if (resultsArea) resultsArea.scrollIntoView({ behavior: 'smooth', block: 'start' });
    }
};

const removeProgram = (id) => {
    selectedProgramIds.value = selectedProgramIds.value.filter(pid => pid !== id);

    // 同步移除檢核結果
    let programName = '';
    for (const collegePrograms of Object.values(programsByCollege.value)) {
        if (collegePrograms[id]) {
            programName = collegePrograms[id].name;
            break;
        }
    }

    if (programName) {
        checkResults.value = checkResults.value.filter(res => res.programName !== programName);
        const maxPage = Math.ceil(checkResults.value.length / pageSize.value) || 1;
        if (currentPage.value > maxPage) {
            currentPage.value = maxPage;
        }
    }
};

const confirmClearPrograms = () => {
    if (confirm('確定要清空所有已選學程嗎？')) {
        selectedProgramIds.value = [];
        checkResults.value = [];
    }
};

const closeAllModals = () => {
    showDisclaimerModal.value = false;
    showPrivacyModal.value = false;
    showTermsModal.value = false;
    showContactModal.value = false;
    showCompletionModal.value = false;
};

const handleKeydown = (e) => {
    if (e.key === 'Escape') {
        closeAllModals();
    }
};

const handleScroll = () => {
    showScrollTop.value = window.scrollY > 300;
};

const scrollToTop = () => {
    window.scrollTo({ top: 0, behavior: 'smooth' });
};

// --- Lifecycle 鉤子 ---
onMounted(() => {
    loadPrograms();
    window.addEventListener('keydown', handleKeydown);
    window.addEventListener('scroll', handleScroll);
});

onUnmounted(() => {
    window.removeEventListener('keydown', handleKeydown);
    window.removeEventListener('scroll', handleScroll);
});
</script>

<template>
    <div class="max-w-5xl mx-auto glass-panel rounded-3xl p-8 sm:p-12 my-8 sm:my-12 animate-entry">
        <h1
            class="text-4xl sm:text-5xl md:text-6xl font-bold text-center text-emerald-700 mb-2 tracking-tight font-serif">
            ProAnalyst
        </h1>
        <p class="text-xl sm:text-2xl font-bold text-center text-stone-700 mb-6 tracking-[0.3em] font-serif">
            個人化學程潛能分析
        </p>
        <p class="text-stone-600 mb-2 text-center text-lg max-w-2xl mx-auto leading-relaxed">
            上傳修課紀錄，即時分析與學程匹配度及修習進度<br>不再錯過任何一個學程的申請，掌握所有通過學程的良機
        </p>
        <p class="text-stone-400 text-xs text-center mb-10">
            （分析師的小秘密🙊：通過的學分學程可以<a href="https://gecenter.nccu.edu.tw/student-zone/gec-credit-waiver/" target="_blank"
                class="underline hover:text-emerald-600 transition-colors">認定成對應領域的通識③學分</a>喔）
        </p>

        <FileUpload :studentFile="studentFile" @file-change="processFile" />

        <div class="flex items-center mb-6">
            <div class="flex-grow border-t border-gray-300"></div>
            <div class="flex items-center mx-4 text-gray-600 font-medium text-center">
                <span class="flex-shrink-0 mr-2">⬇️</span>
                <span
                    class="flex flex-wrap justify-center gap-x-1 sm:gap-x-2 font-normal font-serif text-sm sm:text-lg">
                    <span class="whitespace-nowrap">由分析師 <span class="font-[750]">智慧推薦</span> 最適合的學程</span>
                    <span class="whitespace-nowrap">或是選擇個別學程 <span class="font-[750]">查看修習進度</span></span>
                </span>
                <span class="flex-shrink-0 ml-2">⬇️</span>
            </div>
            <div class="flex-grow border-t border-gray-300"></div>
        </div>

        <!-- 頁籤切換 -->
        <div class="flex justify-center mb-10 animate-entry delay-200">
            <div class="bg-stone-200/50 p-1.5 rounded-2xl inline-flex shadow-inner backdrop-blur-sm">
                <button @click="activeTab = 'recommendation'"
                    class="py-2.5 px-6 sm:px-8 rounded-xl text-sm sm:text-base font-bold transition-all duration-300 focus:outline-none flex items-center gap-2 font-serif tracking-wide"
                    :class="activeTab === 'recommendation' ? 'bg-white text-emerald-900 shadow-md transform scale-105' : 'text-stone-500 hover:text-stone-700'">
                    <span>🔮</span> 智慧學程推薦
                </button>
                <button @click="activeTab = 'check'"
                    class="py-2.5 px-6 sm:px-8 rounded-xl text-sm sm:text-base font-bold transition-all duration-300 focus:outline-none flex items-center gap-2 font-serif tracking-wide"
                    :class="activeTab === 'check' ? 'bg-white text-emerald-900 shadow-md transform scale-105' : 'text-stone-500 hover:text-stone-700'">
                    <span>✏️</span> 個別學程要求
                </button>
            </div>
        </div>

        <!-- 新增：學程推薦區塊 -->
        <div v-if="activeTab === 'recommendation'" class="mb-8 animate-entry delay-300">
            <div
                class="bg-white/60 backdrop-blur-md p-6 sm:p-8 rounded-3xl border border-white/50 shadow-xl shadow-stone-200/40">
                <h2
                    class="text-2xl sm:text-3xl font-bold text-emerald-900 mb-4 flex items-center justify-center font-serif tracking-wide">
                    智慧學程推薦排行榜
                </h2>
                <blockquote
                    class="mb-8 py-4 border-y-2 border-emerald-300 text-stone-500 text-center text-xl italic font-bold font-serif tracking-wider">
                    「你的學分比你想像的更有價值。」
                </blockquote>
                <p class="text-stone-600 mb-6 text-lg leading-relaxed text-center">
                    分析師將交叉比對您的修課紀錄與所有學程標準，找出您已具備資格或高潛力的學程
                </p>

                <button @click="startRecommendation" :disabled="!studentFile || isRecommending"
                    class="group relative mb-8 px-12 py-4 bg-stone-900 font-bold text-lg rounded-xl border border-emerald-500/50 transition-all duration-500 disabled:opacity-50 disabled:cursor-not-allowed flex items-center mx-auto tracking-[0.2em] overflow-hidden"
                    :class="studentFile ? 'text-emerald-400 hover:bg-stone-800 shadow-[0_0_20px_rgba(52,211,153,0.3)] hover:shadow-[0_0_35px_rgba(52,211,153,0.6)] transform hover:-translate-y-1 active:scale-95' : 'text-stone-600'">
                    <div v-if="studentFile"
                        class="absolute inset-0 bg-gradient-to-r from-transparent via-emerald-500/10 to-transparent translate-x-[-100%] group-hover:translate-x-[100%] transition-transform duration-1000 ease-in-out">
                    </div>
                    <span v-if="isRecommending" class="mr-3 relative z-10">
                        <svg class="animate-spin h-5 w-5 text-emerald-400" xmlns="http://www.w3.org/2000/svg"
                            fill="none" viewBox="0 0 24 24">
                            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4">
                            </circle>
                            <path class="opacity-75" fill="currentColor"
                                d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z">
                            </path>
                        </svg>
                    </span>
                    <span class="relative z-10">{{ isRecommending ? '深度分析中...' : '開始智慧推薦' }}</span>
                </button>

                <div v-if="hasRunRecommendation" class="space-y-4 mt-2">
                    <div v-if="recommendationResults.length === 0"
                        class="text-stone-500 text-sm italic text-center py-8">
                        尚無符合推薦門檻的學程
                    </div>
                    <div v-for="rec in rankedRecommendations" :key="rec.programID"
                        class="group bg-white p-6 rounded-2xl border border-stone-100 shadow-sm transition-all duration-300 flex flex-col sm:flex-row sm:items-center justify-between gap-6 relative overflow-hidden">
                        <!-- Decorative background accent -->
                        <div class="absolute top-0 left-0 w-1.5 h-full transition-colors duration-300" :class="{
                            'bg-emerald-600': rec.isCompleted,
                            'bg-stone-300': !rec.isCompleted
                        }">
                        </div>

                        <div class="flex items-center gap-4">
                            <!-- 排名徽章 -->
                            <div class="flex-shrink-0 w-14 h-14 flex items-center justify-center rounded-full font-serif font-bold text-2xl shadow-inner"
                                :class="{
                                    'bg-amber-100 text-amber-700 ring-4 ring-amber-50': rec.rank === 1,
                                    'bg-stone-200 text-stone-600 ring-4 ring-stone-100': rec.rank === 2,
                                    'bg-orange-100 text-orange-700 ring-4 ring-orange-50': rec.rank === 3,
                                    'bg-stone-50 text-stone-400': rec.rank > 3
                                }">
                                {{ rec.rank }}
                            </div>
                            <div>
                                <div
                                    class="font-bold text-stone-800 text-xl group-hover:text-emerald-900 transition-colors font-serif tracking-wide">
                                    {{ rec.programName }}</div>
                                <div class="text-sm text-stone-500 mt-2 flex flex-wrap items-center gap-2">
                                    <span
                                        class="px-2.5 py-1 rounded-md bg-stone-100 text-stone-600 text-xs font-bold tracking-wider uppercase">{{
                                            rec.type === 'micro' ? '微學程' : '學分學程' }}</span>
                                    <span>已修 <span class="font-bold text-stone-700">{{ rec.totalPassedCredits }}</span>
                                        / {{ rec.minCredits }} 學分</span>
                                    <span v-if="rec.passedPrereqCredits > 0"
                                        class="text-emerald-600 font-bold text-xs bg-emerald-50 px-2 py-0.5 rounded-full">
                                        (+ 先修 {{ rec.passedPrereqCredits }} 學分)
                                    </span>
                                </div>
                            </div>
                        </div>
                        <div class="flex items-center gap-4">
                            <div class="text-right">
                                <div class="text-3xl font-bold font-serif text-emerald-700 leading-none" :class="{
                                    'opacity-100': rec.rank === 1,
                                    'opacity-80': rec.rank === 2,
                                    'opacity-60': rec.rank === 3,
                                    'opacity-40': rec.rank === 4,
                                    'opacity-30': rec.rank >= 5
                                }">
                                    <span
                                        class="text-sm font-sans font-bold text-stone-400 mr-1 tracking-wider">剩餘</span>
                                    <span :class="rec.remaining < 0.1 ? 'text-4xl text-teal-600 drop-shadow-sm' : ''">{{
                                        rec.remaining.toFixed(0) }}</span>
                                    <span class="text-base font-sans font-medium text-stone-400 ml-1">學分</span>
                                </div>
                                <div class="text-xs text-stone-400 uppercase tracking-widest font-medium mt-1">
                                    <template v-if="rec.isRestricted">
                                        <span
                                            class="text-stone-600 font-bold inline-block text-left leading-[1.5]">身分限制</span>
                                    </template>
                                    <template v-else-if="rec.isCompleted">
                                        完成度 <span class="ml-1 text-emerald-600 font-bold">100%</span>
                                    </template>
                                    <template v-else-if="rec.completionRate >= 1">
                                        <span
                                            class="text-amber-600 font-bold leading-[1.5]">尚有其他條件未滿足<br>點擊按鈕瞭解更多↗</span>
                                    </template>
                                    <template v-else>
                                        完成度 <span class="ml-1 text-emerald-600 font-bold">{{ (rec.completionRate *
                                            100).toFixed(0) }}%</span>
                                    </template>
                                </div>
                            </div>
                            <button @click="addProgramToSelection(rec.programID, rec.programName)"
                                class="px-6 py-3 bg-emerald-600 text-white hover:bg-emerald-700 rounded-xl text-sm font-bold transition-all shadow-sm hover:shadow-md whitespace-nowrap">
                                還少哪些課？
                            </button>
                        </div>
                    </div>

                    <!-- 前往要求區按鈕 -->
                    <div v-if="hasRunRecommendation" class="mt-8 text-center pt-6 border-t border-stone-100">
                        <button @click="activeTab = 'check'"
                            class="px-8 py-3 bg-emerald-700 hover:bg-emerald-800 text-white font-bold rounded-xl shadow-lg shadow-emerald-900/20 transition-all duration-200 flex items-center justify-center mx-auto transform hover:-translate-y-1">
                            前往確認學程要求
                            <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 ml-2" viewBox="0 0 20 20"
                                fill="currentColor">
                                <path fill-rule="evenodd"
                                    d="M10.293 3.293a1 1 0 011.414 0l6 6a1 1 0 010 1.414l-6 6a1 1 0 01-1.414-1.414L14.586 11H3a1 1 0 110-2h11.586l-4.293-4.293a1 1 0 010-1.414z"
                                    clip-rule="evenodd" />
                            </svg>
                        </button>
                    </div>
                </div>
            </div>
        </div>

        <div v-if="activeTab === 'check'" class="animate-entry delay-300">
            <div class="mb-8 p-6 sm:p-8 border border-stone-200 bg-white rounded-3xl shadow-sm">
                <h2
                    class="text-2xl sm:text-3xl font-bold text-emerald-900 mb-6 flex items-center font-serif tracking-wide">
                    選取欲檢核的學分學程
                </h2>

                <div class="flex flex-col sm:flex-row sm:items-start gap-4 mb-4">
                    <!-- 搜尋列 -->
                    <div class="w-full sm:w-1/2">
                        <label for="programSearch" class="block text-sm font-bold text-stone-700 mb-2">搜尋學程名稱
                            (跨學院搜尋)：</label>
                        <input type="text" id="programSearch" v-model="searchQuery" placeholder="輸入關鍵字..."
                            class="block w-full px-4 py-3 text-base border border-stone-300 focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500 rounded-xl bg-stone-50 transition-shadow">
                    </div>

                    <div class="text-stone-400 font-serif italic shrink-0 self-center sm:self-auto sm:pt-9 px-2">or
                    </div>

                    <!-- 學院選擇下拉選單 -->
                    <div class="w-full sm:w-1/2" :class="{ 'opacity-50 pointer-events-none': searchQuery }">
                        <label for="collegeSelect"
                            class="block text-sm font-bold text-stone-700 mb-2">選擇設置單位或所屬學院：</label>
                        <select id="collegeSelect" v-model="selectedCollege"
                            class="block w-full px-4 py-3 text-base border border-stone-300 focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:border-emerald-500 rounded-xl bg-stone-50 transition-shadow">
                            <option v-for="collegeName in sortedCollegeNames" :key="collegeName" :value="collegeName">{{
                                collegeName }}</option>
                        </select>
                    </div>
                </div>

                <!-- 學程類型選擇 (Radio Buttons) -->
                <div class="flex items-center space-x-6 pb-4 mb-4 border-b border-stone-100">
                    <label class="inline-flex items-center cursor-pointer group">
                        <input type="radio" value="credit" v-model="selectedProgramType"
                            class="h-5 w-5 text-emerald-700 border-stone-300 focus:ring-emerald-500 transition-colors accent-emerald-600">
                        <span
                            class="ml-2 text-stone-700 font-medium group-hover:text-emerald-700 transition-colors">學分學程</span>
                    </label>
                    <label class="inline-flex items-center cursor-pointer group">
                        <input type="radio" value="micro" v-model="selectedProgramType"
                            class="h-5 w-5 text-emerald-700 border-stone-300 focus:ring-emerald-500 transition-colors accent-emerald-600">
                        <span
                            class="ml-2 text-stone-700 font-medium group-hover:text-emerald-700 transition-colors">微學程</span>
                    </label>
                </div>

                <p v-if="selectedProgramType === 'credit'"
                    class="text-sm text-stone-500 mb-6 bg-stone-50 p-3 rounded-lg border border-stone-100">
                    註：學分學程認列科目至少應有三分之一學分數不屬於原學系、所之專業必修科目（此檢核項目尚未建置，請使用者自行確認）
                </p>

                <p v-if="selectedProgramType === 'micro'"
                    class="text-sm text-stone-500 mb-6 bg-stone-50 p-3 rounded-lg border border-stone-100">
                    註：微學程所認列之通識課程以一門為限（以學分較多者計）
                </p>

                <div id="programCheckboxes" class="space-y-6">
                    <!-- 一般學分學程 / 微學程 -->
                    <div>
                        <h3 v-if="selectedProgramType === 'credit' && Object.keys(secondaryPrograms).length > 0"
                            class="text-lg font-bold text-emerald-800 mb-4 flex items-center">
                            <span class="w-1.5 h-6 bg-emerald-500 rounded-full mr-2"></span>
                            校級學分學程
                        </h3>
                        <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
                            <div v-for="(program, id) in primaryPrograms" :key="id"
                                class="flex items-start p-3 rounded-lg hover:bg-stone-50 transition-colors cursor-pointer"
                                @click="!selectedProgramIds.includes(id) ? selectedProgramIds.push(id) : removeProgram(id)">
                                <input type="checkbox" :value="id" v-model="selectedProgramIds"
                                    class="mt-1.5 h-5 w-5 text-emerald-700 border-stone-300 rounded focus:ring-emerald-500 shrink-0 cursor-pointer accent-emerald-600"
                                    @click.stop>
                                <label class="ml-3 text-lg font-bold text-stone-700 cursor-pointer tracking-wide">
                                    {{ program.name }}
                                    <p class="text-xs text-stone-500 mt-1 font-normal leading-relaxed">{{
                                        program.description }}
                                    </p>
                                </label>
                            </div>
                        </div>
                    </div>

                    <!-- 專長學程 (僅在選擇學分學程時顯示) -->
                    <div v-if="Object.keys(secondaryPrograms).length > 0">
                        <h3 class="text-lg font-bold text-emerald-800 mb-4 flex items-center">
                            <span class="w-1.5 h-6 bg-emerald-500 rounded-full mr-2"></span>
                            院級專長學程
                        </h3>
                        <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
                            <div v-for="(program, id) in secondaryPrograms" :key="id"
                                class="flex items-start p-3 rounded-lg hover:bg-stone-50 transition-colors cursor-pointer"
                                @click="!selectedProgramIds.includes(id) ? selectedProgramIds.push(id) : removeProgram(id)">
                                <input type="checkbox" :value="id" v-model="selectedProgramIds"
                                    class="mt-1.5 h-5 w-5 text-emerald-700 border-stone-300 rounded focus:ring-emerald-500 shrink-0 cursor-pointer accent-emerald-600"
                                    @click.stop>
                                <label class="ml-3 text-lg font-bold text-stone-700 cursor-pointer tracking-wide">
                                    {{ program.name }}
                                    <p class="text-xs text-stone-500 mt-1 font-normal leading-relaxed">{{
                                        program.description }}
                                    </p>
                                </label>
                            </div>
                        </div>
                    </div>
                    <div v-if="Object.keys(programsByCollege).length === 0" class="text-sm text-rose-500">
                        載入學程清單中...
                    </div>
                </div>
                <p id="programSelectionStatus" class="mt-4 text-sm text-rose-500 font-bold"
                    v-show="programSelectionStatus">{{
                        programSelectionStatus }}</p>
            </div>

            <div class="mb-8">
                <button id="checkButton" @click="startCheck" :disabled="!isReadyToCheck || isChecking"
                    class="w-full py-4 px-6 bg-emerald-700 hover:bg-emerald-800 text-white font-bold text-lg rounded-xl shadow-xl shadow-emerald-900/20 transition-all duration-200 disabled:opacity-50 disabled:cursor-not-allowed transform active:scale-[0.99]">
                    <span id="buttonText">{{ buttonText }}</span>
                </button>
            </div>

            <div class="mt-12 pt-8 border-t border-stone-200">
                <h2 class="text-3xl font-bold text-emerald-900 mb-8 font-serif text-center tracking-wide">檢核結果報告</h2>
                <div id="resultsArea" class="space-y-6">
                    <p v-if="checkResults.length === 0 && !isChecking" class="text-stone-400 text-center py-10">
                        檢核結果將顯示在此處</p>

                    <!-- 上方分頁控制 -->
                    <div v-if="checkResults.length > pageSize"
                        class="flex flex-col sm:flex-row justify-center items-center gap-4 pb-4">
                        <button @click="changePage(currentPage - 1)" :disabled="currentPage === 1"
                            class="px-6 py-2 bg-white border border-stone-200 text-stone-600 font-bold rounded-xl shadow-sm transition-all hover:bg-stone-50 disabled:opacity-50 disabled:cursor-not-allowed flex items-center">
                            ← 上一頁
                        </button>

                        <span class="text-stone-600 font-bold font-serif text-lg">第 {{ currentPage }} 頁 / 共 {{
                            totalPages }}
                            頁</span>

                        <button @click="changePage(currentPage + 1)" :disabled="currentPage === totalPages"
                            class="px-6 py-2 bg-white border border-stone-200 text-stone-600 font-bold rounded-xl shadow-sm transition-all hover:bg-stone-50 disabled:opacity-50 disabled:cursor-not-allowed flex items-center">
                            下一頁 →
                        </button>
                    </div>

                    <CheckResultCard v-for="result in visibleCheckResults" :key="result.programName" :result="result" />

                    <!-- 分頁控制 -->
                    <div v-if="checkResults.length > pageSize"
                        class="flex flex-col sm:flex-row justify-center items-center gap-4 pt-6 pb-2">
                        <button @click="changePage(currentPage - 1)" :disabled="currentPage === 1"
                            class="px-6 py-2 bg-white border border-stone-200 text-stone-600 font-bold rounded-xl shadow-sm transition-all hover:bg-stone-50 disabled:opacity-50 disabled:cursor-not-allowed flex items-center">
                            ← 上一頁
                        </button>

                        <span class="text-stone-600 font-bold font-serif text-lg">第 {{ currentPage }} 頁 / 共 {{
                            totalPages }}
                            頁</span>

                        <button @click="changePage(currentPage + 1)" :disabled="currentPage === totalPages"
                            class="px-6 py-2 bg-white border border-stone-200 text-stone-600 font-bold rounded-xl shadow-sm transition-all hover:bg-stone-50 disabled:opacity-50 disabled:cursor-not-allowed flex items-center">
                            下一頁 →
                        </button>
                    </div>
                </div>
            </div>
        </div>

        <!-- Footer -->
        <footer class="mt-16 pt-8 border-t border-stone-200 text-center text-sm text-stone-400 pb-8">
            <div class="flex justify-center space-x-4 mb-3">
                <button @click="showPrivacyModal = true"
                    class="hover:text-emerald-600 transition-colors font-medium">隱私權政策</button>
                <span class="text-stone-300">|</span>
                <button @click="showTermsModal = true"
                    class="hover:text-emerald-600 transition-colors font-medium">服務條款</button>
            </div>
            <div class="flex justify-center mb-2">
                <img src="./assets/logo_transparent.svg" alt="ProAnalyst Logo"
                    class="h-8 w-auto transition-opacity duration-300" />
            </div>
            <p class="mb-2">&copy; {{ new Date().getFullYear() }} 𤫹焈焈麀普羅安納利斯特團隊. Licensed under the MIT License.</p>
        </footer>
    </div>

    <AppModals v-model:showDisclaimer="showDisclaimerModal" :disclaimerAction="disclaimerAction"
        @confirmDisclaimer="disclaimerAction === 'check' ? executeCheck() : executeRecommendation()"
        v-model:showCompletion="showCompletionModal" :completedPrograms="completedPrograms"
        v-model:showContact="showContactModal" v-model:showPrivacy="showPrivacyModal"
        v-model:showTerms="showTermsModal" />

    <!-- Selected Programs Sidebar -->
    <div class="fixed inset-0 z-50 pointer-events-none overflow-hidden"
        v-if="activeTab === 'check' || hasRunRecommendation">
        <!-- Sidebar Panel -->
        <div class="absolute top-24 bottom-48 right-0 flex max-w-full pl-10 pointer-events-none">
            <div class="pointer-events-auto w-full max-w-md relative transform transition-transform duration-300 ease-in-out"
                :class="showSelectedSidebar ? 'translate-x-0' : 'translate-x-full'">
                <!-- Toggle Button (Combined Trigger & Retract) -->
                <button @click="showSelectedSidebar = !showSelectedSidebar"
                    class="absolute top-[calc(33vh-6rem)] -left-10 bg-white border-l-4 border-emerald-600 shadow-lg py-4 px-1 rounded-l-lg hover:bg-stone-50 transition-all flex flex-col items-center gap-2 z-50 w-10">
                    <span
                        class="writing-vertical-rl text-emerald-800 font-black tracking-widest text-sm py-1 font-serif"
                        style="writing-mode: vertical-rl;">已選學程</span>
                    <span
                        class="bg-emerald-600 text-white text-xs font-bold rounded-full w-6 h-6 flex items-center justify-center shadow-sm font-serif">
                        {{ selectedProgramsList.length }}
                    </span>
                </button>

                <div
                    class="flex h-full flex-col overflow-y-scroll bg-white shadow-2xl border border-stone-200 rounded-l-2xl">
                    <div class="relative pt-8 flex-1 px-4 sm:px-6">
                        <ul role="list" class="space-y-3">
                            <li v-for="p in selectedProgramsList" :key="p.id" @click="removeProgram(p.id)"
                                class="bg-stone-50 rounded-xl p-4 border border-stone-100 shadow-sm flex justify-between items-center group hover:border-rose-200 hover:bg-rose-50 transition-colors cursor-pointer">
                                <span class="font-bold text-stone-700 group-hover:text-rose-700 transition-colors">{{
                                    p.name }}</span>
                                <button
                                    class="text-stone-400 hover:text-rose-600 p-1 rounded-full hover:bg-rose-100 transition-colors">
                                    <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20"
                                        fill="currentColor">
                                        <path fill-rule="evenodd"
                                            d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
                                            clip-rule="evenodd" />
                                    </svg>
                                </button>
                            </li>
                        </ul>
                        <div v-if="selectedProgramsList.length === 0"
                            class="text-center text-stone-400 mt-10 flex flex-col items-center">
                            <span class="text-4xl mb-2">📭</span>
                            <p>尚未選擇任何學程</p>
                        </div>
                    </div>
                    <div class="border-t border-stone-200 p-4 sm:px-6 bg-stone-50/50 flex flex-col gap-3">
                        <button v-if="activeTab === 'recommendation' && selectedProgramsList.length > 0"
                            @click="activeTab = 'check'"
                            class="w-full py-3 bg-emerald-600 hover:bg-emerald-700 text-white font-bold rounded-xl shadow-sm transition-colors flex items-center justify-center gap-2">
                            <span>✏️</span> 前往確認學程要求
                        </button>
                        <button @click="confirmClearPrograms" v-if="selectedProgramsList.length > 0"
                            class="w-full py-2 text-stone-400 hover:text-rose-500 text-sm font-bold transition-colors">
                            清空所有已選學程
                        </button>
                    </div>
                </div>
            </div>
        </div>
    </div>

    <!-- Scroll to Top Button -->
    <Transition enter-active-class="transition ease-out duration-300"
        enter-from-class="transform opacity-0 translate-y-4" enter-to-class="transform opacity-100 translate-y-0"
        leave-active-class="transition ease-in duration-200" leave-from-class="transform opacity-100 translate-y-0"
        leave-to-class="transform opacity-0 translate-y-4">
        <button v-if="showScrollTop" @click="scrollToTop"
            class="fixed bottom-28 right-8 z-40 bg-white text-emerald-700 p-3 rounded-full shadow-lg border border-emerald-100 hover:bg-emerald-50 transition-all transform hover:scale-110 flex items-center justify-center group"
            title="回到頂部">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24"
                stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 10l7-7m0 0l7 7m-7-7v18" />
            </svg>
            <span
                class="absolute right-full mr-3 bg-stone-800 text-white text-xs px-3 py-1.5 rounded-lg opacity-0 group-hover:opacity-100 transition-opacity whitespace-nowrap font-bold shadow-lg">
                回到頂部
            </span>
        </button>
    </Transition>

    <!-- Floating Contact Button -->
    <button @click="showContactModal = true"
        class="fixed bottom-8 right-8 z-40 bg-emerald-700 hover:bg-emerald-800 text-white p-4 rounded-full shadow-2xl shadow-emerald-900/30 transition-transform transform hover:scale-110 flex items-center justify-center group"
        title="聯絡我們">
        <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                d="M8 10h.01M12 10h.01M16 10h.01M9 16H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-5l-5 5v-5z" />
        </svg>
        <span
            class="absolute right-full mr-3 bg-stone-800 text-white text-xs px-3 py-1.5 rounded-lg opacity-0 group-hover:opacity-100 transition-opacity whitespace-nowrap font-bold shadow-lg">
            聯絡我們
        </span>
    </button>

    <!-- 通知 Toast -->
    <Transition enter-active-class="transition ease-out duration-300"
        enter-from-class="transform opacity-0 translate-x-4" enter-to-class="transform opacity-100 translate-x-0"
        leave-active-class="transition ease-in duration-200" leave-from-class="transform opacity-100 translate-x-0"
        leave-to-class="transform opacity-0 translate-x-4">
        <div v-if="notification.show"
            class="fixed top-24 right-4 z-50 max-w-sm w-full bg-white/90 backdrop-blur border-l-4 shadow-2xl rounded-r-xl pointer-events-auto"
            :class="notification.type === 'success' ? 'border-emerald-500' : 'border-emerald-500'">
            <div class="p-4 flex items-start">
                <div class="flex-shrink-0">
                    <svg v-if="notification.type === 'success'" class="h-6 w-6 text-emerald-500" fill="none"
                        viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
                    </svg>
                    <svg v-else class="h-6 w-6 text-emerald-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                            d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                    </svg>
                </div>
                <div class="ml-3 w-0 flex-1 pt-0.5">
                    <p class="text-sm font-bold text-stone-800">{{ notification.message }}</p>
                    <button v-if="notification.action" @click="notification.action.handler"
                        class="mt-2 text-sm font-bold text-emerald-600 hover:text-emerald-500 focus:outline-none underline">
                        {{ notification.action.text }}
                    </button>
                </div>
                <div class="ml-4 flex-shrink-0 flex self-start">
                    <button @click="notification.show = false"
                        class="rounded-md inline-flex text-stone-400 hover:text-stone-500 focus:outline-none"
                        title="關閉">
                        <span class="sr-only">Close</span>
                        <svg class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
                            <path fill-rule="evenodd"
                                d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
                                clip-rule="evenodd" />
                        </svg>
                    </button>
                </div>
            </div>
        </div>
    </Transition>
</template>
