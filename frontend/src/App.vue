<script lang="ts" setup>
import {
  NConfigProvider,
  NGlobalStyle,
  NMessageProvider,
  NDialogProvider,
  darkTheme,
} from "naive-ui";
import MainLayout from "./components/MainLayout.vue";

const themeOverrides = {
  common: {
    primaryColor: "#38bdf8",
    primaryColorHover: "#0ea5e9",
  },
  Card: {
    borderColor: "#334155",
  },
};
</script>

<template>
  <n-config-provider :theme="darkTheme" :theme-overrides="themeOverrides">
    <n-global-style />
    <n-dialog-provider>
      <n-message-provider>
        <MainLayout />
      </n-message-provider>
    </n-dialog-provider>
  </n-config-provider>
</template>

<style>
body,
#app,
.n-config-provider {
  font-family: "Noto Sans", "Roboto", "Segoe UI", sans-serif !important;
}
</style>
