<script lang="ts" setup>
// --- Vue 核心 ---
import { onMounted, ref, watch } from "vue";
import { useI18n } from "vue-i18n";

// --- Wails & 后端绑定 ---
import { Dialogs } from "@wailsio/runtime";
import {
  GetSavePath,
  SetSavePath,
  GetHostName,
  SetHostName,
  GetAutoAccept,
  SetAutoAccept,
  GetSaveHistory,
  SetSaveHistory,
  GetVersion,
  GetLanguage,
  SetLanguage,
  SetCloseToSystray,
  GetCloseToSystray,
} from "../../bindings/mesh-drop/internal/config/config";
import { Language } from "bindings/mesh-drop/internal/config";

// --- 状态 ---
const savePath = ref("");
const hostName = ref("");
const autoAccept = ref(false);
const saveHistory = ref(false);
const version = ref("");
const closeToSystray = ref(false);

const { t, locale } = useI18n();

const languages = [
  { title: "English", value: "en" },
  { title: "简体中文", value: "zh-Hans" },
];

// ---生命周期 ---
onMounted(async () => {
  savePath.value = await GetSavePath();
  hostName.value = await GetHostName();
  autoAccept.value = await GetAutoAccept();
  saveHistory.value = await GetSaveHistory();
  version.value = await GetVersion();
  let l = await GetLanguage();
  if (l != "") {
    locale.value = l;
  }
  closeToSystray.value = await GetCloseToSystray();
});

// --- 方法 ---
const changeSavePath = async () => {
  const opts: Dialogs.OpenFileDialogOptions = {
    Title: t("settings.selectSavePath"),
    CanChooseDirectories: true,
    CanChooseFiles: false,
    AllowsMultipleSelection: false,
  };
  const path = await Dialogs.OpenFile(opts);
  if (path && typeof path === "string") {
    await SetSavePath(path);
    savePath.value = path;
  }
};

// 监听语言变化
watch(locale, async (newVal) => {
  await SetLanguage(newVal as Language);
});
</script>

<template>
  <v-list lines="one" bg-color="transparent">
    <!-- 保存路径 -->
    <v-list-item :title="t('settings.savePath')" :subtitle="savePath">
      <template #prepend>
        <v-icon icon="mdi-folder-download"></v-icon>
      </template>
      <template #append>
        <v-btn
          variant="text"
          color="primary"
          @click="changeSavePath"
          prepend-icon="mdi-pencil"
        >
          {{ t("settings.change") }}
        </v-btn>
      </template>
    </v-list-item>

    <!-- 主机名 -->
    <v-list-item :title="t('settings.hostName')">
      <template #prepend>
        <v-icon icon="mdi-laptop"></v-icon>
      </template>
      <template #append>
        <v-text-field
          clearable
          variant="underlined"
          v-model="hostName"
          width="200"
          @update:modelValue="SetHostName"
        ></v-text-field>
      </template>
    </v-list-item>

    <!-- 保存历史 -->
    <v-list-item :title="t('settings.saveHistory')">
      <template #prepend>
        <v-icon icon="mdi-history"></v-icon>
      </template>
      <template #append>
        <v-switch
          v-model="saveHistory"
          color="primary"
          inset
          hide-details
          @update:modelValue="SetSaveHistory(saveHistory)"
        ></v-switch>
      </template>
    </v-list-item>

    <!-- 自动接受 -->
    <v-list-item :title="t('settings.autoAccept')">
      <template #prepend>
        <v-icon icon="mdi-content-save"></v-icon>
      </template>
      <template #append>
        <v-switch
          v-model="autoAccept"
          color="primary"
          inset
          hide-details
          @update:modelValue="SetAutoAccept(autoAccept)"
        ></v-switch>
      </template>
    </v-list-item>

    <!-- 关闭窗口时最小化到托盘 -->
    <v-list-item :title="t('settings.closeToSystray')">
      <template #prepend>
        <v-icon icon="mdi-tray"></v-icon>
      </template>
      <template #append>
        <v-switch
          v-model="closeToSystray"
          color="primary"
          inset
          hide-details
          @update:modelValue="SetCloseToSystray(closeToSystray)"
        ></v-switch>
      </template>
    </v-list-item>

    <!-- 语言 -->
    <v-list-item :title="t('settings.language')">
      <template #prepend>
        <v-icon icon="mdi-translate"></v-icon>
      </template>
      <template #append>
        <v-select
          v-model="locale"
          :items="languages"
          variant="underlined"
          density="compact"
          hide-details
          width="150"
        ></v-select>
      </template>
    </v-list-item>

    <!-- 版本 -->
    <v-list-item :title="t('settings.version')">
      <template #prepend>
        <v-icon icon="mdi-information"></v-icon>
      </template>
      <template #append>
        <div class="text-grey">{{ version }}</div>
      </template>
    </v-list-item>
  </v-list>
</template>
