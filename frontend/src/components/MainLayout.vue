<script lang="ts" setup>
import { onMounted, ref, computed, h } from "vue";
import PeerCard from "./PeerCard.vue";
import TransferItem from "./TransferItem.vue";
import {
  NLayout,
  NLayoutHeader,
  NLayoutContent,
  NLayoutSider,
  NSpace,
  NText,
  NEmpty,
  NMenu,
  NBadge,
  NButton,
  NIcon,
} from "naive-ui";
import { FontAwesomeIcon } from "@fortawesome/vue-fontawesome";
import {
  faSatelliteDish,
  faInbox,
  faBars,
  faXmark,
} from "@fortawesome/free-solid-svg-icons";
import { type MenuOption } from "naive-ui";
import { Peer } from "../../bindings/mesh-drop/internal/discovery/models";
import { Transfer } from "../../bindings/mesh-drop/internal/transfer";
import { GetPeers } from "../../bindings/mesh-drop/internal/discovery/service";
import { Events } from "@wailsio/runtime";
import { GetTransferList } from "../../bindings/mesh-drop/internal/transfer/service";

const peers = ref<Peer[]>([]);
const transferList = ref<Transfer[]>([]);
const activeKey = ref("discover");
const showMobileMenu = ref(false);
const isMobile = ref(false);

// 监听窗口大小变化更新 isMobile
onMounted(async () => {
  checkMobile();
  window.addEventListener("resize", checkMobile);
  const list = await GetTransferList();
  transferList.value = (
    (list || []).filter((t) => t !== null) as Transfer[]
  ).sort((a, b) => b.create_time - a.create_time);
});

const checkMobile = () => {
  isMobile.value = window.innerWidth < 768;
  if (!isMobile.value) showMobileMenu.value = false;
};

// --- 菜单选项 ---
const renderIcon = (icon: any) => {
  return () => h(NIcon, null, { default: () => h(FontAwesomeIcon, { icon }) });
};

const menuOptions = computed<MenuOption[]>(() => [
  {
    label: "Discover",
    key: "discover",
    icon: renderIcon(faSatelliteDish),
  },
  {
    label: () =>
      h(
        "div",
        {
          style:
            "display: flex; align-items: center; justify-content: space-between; width: 100%",
        },
        [
          "Transfers",
          pendingCount.value > 0 ?
            h(NBadge, {
              style: "display: inline-flex; align-items: center",
              value: pendingCount.value,
              max: 99,
              type: "error",
            })
          : null,
        ],
      ),
    key: "transfers",
    icon: renderIcon(faInbox),
  },
]);

// --- 后端集成 ---
onMounted(async () => {
  peers.value = await GetPeers();
});

// --- 事件监听 ---

Events.On("peers:update", (event) => {
  peers.value = event.data;
});

Events.On("transfer:refreshList", async () => {
  const list = await GetTransferList();
  transferList.value = (
    (list || []).filter((t) => t !== null) as Transfer[]
  ).sort((a, b) => b.create_time - a.create_time);
});

// --- 计算属性 ---
const pendingCount = computed(() => {
  return transferList.value.filter(
    (t) => t.type === "receive" && t.status === "pending",
  ).length;
});

// --- 操作 ---

const handleMenuUpdate = (key: string) => {
  activeKey.value = key;
  showMobileMenu.value = false;
};
</script>

<template>
  <!-- 小尺寸头部 -->
  <n-layout-header v-if="isMobile" bordered class="mobile-header">
    <n-space
      align="center"
      justify="space-between"
      style="height: 100%; padding: 0 16px">
      <n-text class="logo">Mesh Drop</n-text>
      <n-button
        text
        style="font-size: 24px"
        @click="showMobileMenu = !showMobileMenu">
        <n-icon>
          <FontAwesomeIcon :icon="showMobileMenu ? faXmark : faBars" />
        </n-icon>
      </n-button>
    </n-space>
  </n-layout-header>

  <!-- 小尺寸抽屉菜单 -->
  <n-drawer
    v-model:show="showMobileMenu"
    placement="top"
    height="200"
    v-if="isMobile">
    <n-drawer-content>
      <n-menu
        :value="activeKey"
        :options="menuOptions"
        @update:value="handleMenuUpdate" />
    </n-drawer-content>
  </n-drawer>

  <n-layout
    has-sider
    position="absolute"
    :style="{ top: isMobile ? '64px' : '0' }">
    <!-- 桌面端侧边栏 -->
    <n-layout-sider
      v-if="!isMobile"
      bordered
      width="240"
      content-style="padding: 24px;">
      <div class="desktop-logo">
        <n-text class="logo">Mesh Drop</n-text>
      </div>
      <n-menu
        :value="activeKey"
        :options="menuOptions"
        @update:value="handleMenuUpdate" />
    </n-layout-sider>

    <n-layout-content class="content">
      <div class="content-container">
        <!-- 发现页视图 -->
        <div v-show="activeKey === 'discover'">
          <n-space vertical size="large" v-if="peers.length > 0">
            <div class="peer-grid">
              <div v-for="peer in peers" :key="peer.id">
                <PeerCard
                  :peer="peer"
                  @transferStarted="activeKey = 'transfers'" />
              </div>
            </div>
          </n-space>

          <div v-else class="empty-state">
            <n-empty description="Scanning for peers...">
              <template #icon>
                <n-icon class="radar-icon">
                  <FontAwesomeIcon :icon="faSatelliteDish" />
                </n-icon>
              </template>
            </n-empty>
          </div>
        </div>

        <!-- 传输列表视图 -->
        <div v-show="activeKey === 'transfers'">
          <div v-if="transferList.length > 0">
            <TransferItem
              v-for="transfer in transferList"
              :key="transfer.id"
              :transfer="transfer" />
          </div>
          <div v-else class="empty-state">
            <n-empty style="user-select: none" description="No transfers yet">
              <template #icon>
                <n-icon>
                  <FontAwesomeIcon :icon="faInbox" />
                </n-icon>
              </template>
            </n-empty>
          </div>
        </div>
      </div>
    </n-layout-content>
  </n-layout>
</template>

<style scoped>
.mobile-header {
  height: 64px;
  z-index: 1000;
}

.desktop-logo {
  margin-bottom: 24px;
  padding-left: 8px;
}

.logo {
  font-size: 1.25rem;
  font-weight: 700;
  color: #38bdf8;
}

.content-container {
  padding: 24px;
}

.empty-state {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 90vh;
}

.radar-icon {
  animation: spin 3s linear infinite;
  color: #38bdf8;
  opacity: 0.5;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.peer-grid {
  display: grid;
  grid-template-columns: repeat(1, minmax(0, 1fr));
  gap: 16px;
}

@media (min-width: 500px) {
  .peer-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (min-width: 700px) {
  .peer-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}
</style>
