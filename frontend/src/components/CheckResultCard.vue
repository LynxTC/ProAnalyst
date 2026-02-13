<script setup>
import { ref, computed } from 'vue';

const props = defineProps({
    result: {
        type: Object,
        required: true
    }
});

const isOpen = ref(false);

const toggleDetails = () => {
    isOpen.value = !isOpen.value;
};

const progressPercentage = computed(() => {
    if (!props.result.minRequiredCredits) return 0;
    const pct = (props.result.totalPassedCredits / props.result.minRequiredCredits) * 100;
    return Math.min(pct, 100);
});

const processedCategoryResults = computed(() => {
    if (!props.result.categoryResults) return [];

    return props.result.categoryResults.map(cat => {
        let shouldHide = cat.hideCourseList;

        // 金融科技專長學程：隱藏非標準類別（如群A+群B檢核）的課程列表
        if (props.result.programName === '金融科技專長學程') {
            const standardCategories = ['群A：資訊課程', '群B：金融課程', '選修C：金融科技課程'];
            if (!standardCategories.includes(cat.category)) {
                shouldHide = true;
            }
        }

        // 跨領域精準健康學分學程：隱藏非標準類別（如群組檢核）的課程列表
        if (props.result.programName === '跨領域精準健康學分學程') {
            const standardCategories = ['必修課程', '群A', '群B', '群C', '群D'];
            if (!standardCategories.includes(cat.category)) {
                shouldHide = true;
            }
        }

        // 人力資源管理學程：隱藏總門數檢核類別（該類別僅為計數檢查，無獨立課程列表）
        if (props.result.programName && props.result.programName.includes('人力資源管理學程') && cat.category === '程序課程總門數檢核') {
            shouldHide = true;
        }

        return { ...cat, hideCourseList: shouldHide };
    });
});
</script>

<template>
    <div class="group relative bg-white rounded-2xl border border-stone-200 overflow-hidden">

        <!-- Status Indicator Strip -->
        <div class="absolute left-0 top-0 bottom-0 w-1.5 transition-colors duration-300"
            :class="result.isCompleted ? 'bg-emerald-500' : 'bg-rose-400'"></div>

        <!-- Main Summary Card (Clickable) -->
        <div class="p-6 select-none relative z-10">
            <div class="flex justify-between items-start gap-4">
                <div class="flex-1">
                    <div class="flex items-center gap-2 mb-1">
                        <span v-if="result.type === 'micro'"
                            class="px-2 py-0.5 rounded text-sm font-bold uppercase tracking-wider bg-stone-100 text-stone-500">微學程</span>
                        <h3 class="text-xl font-bold font-serif text-stone-800 leading-tight">
                            {{ result.programName }}
                        </h3>
                    </div>
                    <p class="text-sm text-stone-500 font-medium">{{ result.programDescription }}</p>
                </div>

                <!-- Status Icon -->
                <div class="flex-shrink-0">
                    <div class="px-3 py-1.5 rounded-full flex items-center justify-center gap-2 transition-colors duration-300"
                        :class="result.isCompleted ? 'bg-emerald-100 text-emerald-600' : 'bg-rose-50 text-rose-500'">
                        <span class="font-bold">{{ result.isCompleted ? '✓' : '!' }}</span>
                        <span class="font-bold text-sm">{{ result.isCompleted ? '已修畢' : '未修畢' }}</span>
                    </div>
                </div>
            </div>

            <!-- Progress Bar -->
            <div class="mt-6">
                <div class="flex justify-between text-sm font-bold uppercase tracking-wider text-stone-400 mb-2">
                    <span>修習進度</span>
                    <span class="font-mono" :class="result.isCompleted ? 'text-emerald-600' : 'text-stone-600'">
                        {{ result.totalPassedCredits }} / {{ result.minRequiredCredits }} 學分
                    </span>
                </div>
                <div class="h-2 w-full bg-stone-100 rounded-full overflow-hidden">
                    <div class="h-full transition-all duration-1000 ease-out rounded-full relative"
                        :class="result.isCompleted ? 'bg-emerald-500' : 'bg-stone-800'"
                        :style="{ width: `${progressPercentage}%` }">
                    </div>
                </div>
            </div>

            <!-- Expand/Collapse Indicator -->
            <div @click="toggleDetails" class="flex justify-center mt-4 -mb-2 cursor-pointer">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 text-stone-400 transition-transform duration-300"
                    :class="{ 'rotate-180': isOpen }" viewBox="0 0 20 20" fill="currentColor">
                    <path fill-rule="evenodd"
                        d="M5.293 7.293a1 1 0 011.414 0L10 10.586l3.293-3.293a1 1 0 111.414 1.414l-4 4a1 1 0 01-1.414 0l-4-4a1 1 0 010-1.414z"
                        clip-rule="evenodd" />
                </svg>
            </div>
        </div>

        <!-- Detailed Breakdown (Collapsible) -->
        <div v-show="isOpen" class="bg-stone-50/80 border-t border-stone-100 p-6 animate-fade-in">

            <!-- Restrictions Alert -->
            <div v-if="result.restrictionMessage"
                class="mb-6 p-4 bg-rose-50 border border-rose-100 rounded-xl flex gap-3 items-start">
                <span class="text-lg">🚫</span>
                <div>
                    <h4 class="font-bold text-rose-800 text-sm uppercase tracking-wider mb-1">資格限制</h4>
                    <p class="text-rose-700 text-sm leading-relaxed">{{ result.restrictionMessage }}</p>
                </div>
            </div>

            <!-- Average Score Check -->
            <div v-if="result.avgScoreRequired && result.totalCreditsMet"
                class="mb-6 p-4 bg-white border border-stone-200 rounded-xl shadow-sm flex justify-between items-center">
                <div>
                    <span class="font-bold text-stone-700 text-sm">平均成績檢核</span>
                    <span class="text-sm text-stone-500 ml-2">(標準: {{ result.avgScoreThreshold }} 分)</span>
                </div>
                <div>
                    <span class="font-mono font-bold text-lg"
                        :class="result.avgScoreMet ? 'text-emerald-600' : 'text-rose-600'">
                        {{ result.avgScore }}
                    </span>
                    <span class="text-sm font-bold ml-1"
                        :class="result.avgScoreMet ? 'text-emerald-600' : 'text-rose-600'">分</span>
                </div>
            </div>

            <!-- Categories Grid -->
            <div class="space-y-4">
                <div v-for="cat in processedCategoryResults" :key="cat.category"
                    class="bg-white rounded-xl border border-stone-200 overflow-hidden shadow-sm">

                    <!-- Category Header -->
                    <div class="p-4 flex justify-between items-center bg-stone-50/50 border-b border-stone-100">
                        <h4 class="font-bold text-stone-800 text-sm">{{ cat.category }}</h4>
                        <div class="flex items-center gap-2">
                            <span v-if="cat.limitExceeded"
                                class="text-sm font-bold text-amber-600 bg-amber-50 px-2 py-1 rounded border border-amber-100">
                                ⚠️ {{ cat.exceededMessage }}
                            </span>
                            <span v-if="cat.requiredCount > 0 || cat.requiredCredits > 0"
                                class="px-2 py-1 rounded text-sm font-bold"
                                :class="cat.isMet ? 'bg-emerald-100 text-emerald-800' : 'bg-rose-100 text-rose-800'">
                                {{ cat.isMet ? '已通過' : '未通過' }}
                            </span>
                        </div>
                    </div>

                    <!-- Category Content -->
                    <div class="p-4">
                        <div class="flex justify-between text-sm font-mono text-stone-500 mb-3">
                            <span>
                                <span v-if="cat.requiredCount > 0">要求: {{ cat.requiredCount }} 門</span>
                                <span v-if="cat.requiredCredits > 0" class="ml-2">要求: {{ cat.requiredCredits }}
                                    學分</span>
                            </span>
                            <span
                                :class="((cat.requiredCount > 0 || cat.requiredCredits > 0) && cat.isMet) ? 'text-emerald-600 font-bold' : 'text-stone-800'">
                                {{ cat.passedCredits.toFixed(1) }} 學分 / {{ cat.passedCount }} 門
                            </span>
                        </div>

                        <!-- Course List -->
                        <ul v-if="!cat.hideCourseList" class="space-y-1">
                            <li v-if="cat.passedCourses.length === 0" class="text-sm text-stone-400 italic py-1">
                                無符合課程
                            </li>
                            <li v-for="c in cat.passedCourses" :key="c.name + c.semester"
                                class="flex justify-between items-center text-sm py-1.5 px-2 rounded hover:bg-stone-50 transition-colors group/item">
                                <span class="font-medium text-stone-700 truncate pr-2">{{ c.name }}</span>
                                <div
                                    class="flex items-center gap-2 shrink-0 opacity-70 group-hover/item:opacity-100 transition-opacity">
                                    <span class="font-mono text-sm text-stone-500"><span
                                            class="font-bold text-stone-700">{{ c.credit }} 學分</span> / {{ c.score
                                            }}</span>
                                </div>
                            </li>
                        </ul>
                    </div>
                </div>
            </div>

            <!-- In Progress Section -->
            <div v-if="result.inProgressCourses && result.inProgressCourses.length > 0" class="mt-6">
                <div class="flex items-center gap-2 mb-3">
                    <div class="h-px flex-1 bg-amber-200"></div>
                    <span class="text-sm font-bold text-amber-600 uppercase tracking-widest">修習中課程</span>
                    <div class="h-px flex-1 bg-amber-200"></div>
                </div>
                <div class="bg-amber-50 rounded-xl border border-amber-100 p-1">
                    <div v-for="c in result.inProgressCourses" :key="c.name"
                        class="flex justify-between items-center p-2 text-sm text-amber-900/70 hover:bg-amber-100/50 rounded-lg transition-colors">
                        <span class="font-medium">{{ c.name }}</span>
                        <span class="font-mono text-sm">{{ c.credit }} 學分</span>
                    </div>
                </div>
            </div>

        </div>
    </div>
</template>

<style scoped>
.animate-fade-in {
    animation: fadeIn 0.3s ease-out forwards;
}

@keyframes fadeIn {
    from {
        opacity: 0;
        transform: translateY(-10px);
    }

    to {
        opacity: 1;
        transform: translateY(0);
    }
}
</style>
