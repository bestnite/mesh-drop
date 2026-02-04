<script lang="ts" setup>
// --- Vue 核心 ---
import { onMounted, ref } from "vue";

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
} from "../../bindings/mesh-drop/internal/config/config";

// --- 状态 ---
const savePath = ref("");
const hostName = ref("");
const autoAccept = ref(false);
const saveHistory = ref(false);
const version = ref("");

// ---生命周期 ---
onMounted(async () => {
  savePath.value = await GetSavePath();
  hostName.value = await GetHostName();
  autoAccept.value = await GetAutoAccept();
  saveHistory.value = await GetSaveHistory();
  version.value = await GetVersion();
});

// --- 方法 ---
const changeSavePath = async () => {
  const opts: Dialogs.OpenFileDialogOptions = {
    Title: "Select Save Path",
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
</script>

<template>
  <v-list lines="one" bg-color="transparent">
    <v-list-item title="Save Path" :subtitle="savePath">
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
          Change
        </v-btn>
      </template>
    </v-list-item>
    <v-list-item title="HostName">
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
    <v-list-item title="Save History">
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
    <v-list-item title="Auto Accept">
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
    <v-list-item title="Version">
      <template #prepend>
        <v-icon icon="mdi-information"></v-icon>
      </template>
      <template #append>
        <div class="text-grey">{{ version }}</div>
      </template>
    </v-list-item>
  </v-list>
</template>
