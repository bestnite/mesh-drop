<script lang="ts" setup>
// --- Vue 核心 ---
import { onMounted, ref, computed } from "vue";
import { useI18n } from "vue-i18n";

// --- 组件 ---
import PeerCard from "./PeerCard.vue";
import TransferItem from "./TransferItem.vue";
import SettingsView from "./SettingsView.vue";

// --- 类型 & 模型 ---
import { Peer } from "../../bindings/mesh-drop/internal/discovery/models";
import { Transfer } from "../../bindings/mesh-drop/internal/transfer";

// --- Service & 后端绑定 ---
import { Events } from "@wailsio/runtime";
import { GetPeers } from "../../bindings/mesh-drop/internal/discovery/service";
import {
  GetTransferList,
  CleanFinishedTransferList,
} from "../../bindings/mesh-drop/internal/transfer/service";

// --- 状态 ---
const peers = ref<Peer[]>([]);
const transferList = ref<Transfer[]>([]);
const activeKey = ref("discover");
const drawer = ref(true);
const isMobile = ref(false);
const { t } = useI18n();

// --- 计算属性 ---
const pendingCount = computed(() => {
  return transferList.value.filter(
    (t) => t.type === "receive" && t.status === "pending",
  ).length;
});

const menuItems = computed(() => [
  {
    title: t("menu.discover"),
    value: "discover",
    icon: "mdi-radar",
  },
  {
    title: t("menu.transfers"),
    value: "transfers",
    icon: "mdi-inbox",
    badge: pendingCount.value > 0 ? pendingCount.value : null,
  },
  {
    title: t("menu.settings"),
    value: "settings",
    icon: "mdi-cog",
  },
]);

// --- 生命周期 ---
onMounted(async () => {
  checkMobile();
  window.addEventListener("resize", checkMobile);
  transferList.value = (await GetTransferList()) as Transfer[];

  if (isMobile.value) {
    drawer.value = false;
  }
});

// --- 后端集成 & 事件监听 ---
onMounted(async () => {
  peers.value = await GetPeers();
});

Events.On("peers:update", (event) => {
  peers.value = event.data;
});

Events.On("transfer:refreshList", async () => {
  transferList.value = (await GetTransferList()) as Transfer[];
});

// --- 方法 ---
const checkMobile = () => {
  const mobile = window.innerWidth < 768;
  if (mobile !== isMobile.value) {
    isMobile.value = mobile;
    drawer.value = !mobile;
  }
};

const handleMenuClick = (key: string) => {
  activeKey.value = key;
  if (isMobile.value) {
    drawer.value = false;
  }
};

const handleCleanFinished = async () => {
  await CleanFinishedTransferList();
};
</script>

<template>
  <v-layout>
    <!-- 小屏幕抽屉 -->
    <v-app-bar v-if="isMobile" border flat>
      <v-toolbar-title class="text-primary font-weight-bold"
        >Mesh Drop</v-toolbar-title
      >
      <template #append>
        <v-btn icon="mdi-menu" @click="drawer = !drawer"></v-btn>
      </template>
    </v-app-bar>

    <!-- 导航抽屉 -->
    <v-navigation-drawer v-model="drawer" :permanent="!isMobile">
      <div class="pa-4 d-flex align-center justify-center" v-if="!isMobile">
        <div class="text-h6 text-primary font-weight-bold">Mesh Drop</div>
      </div>

      <v-list nav>
        <v-list-item
          v-for="item in menuItems"
          :key="item.value"
          :value="item.value"
          :active="activeKey === item.value"
          @click="handleMenuClick(item.value)"
          rounded="xl"
          color="primary"
        >
          <template #prepend>
            <v-icon :icon="item.icon"></v-icon>
          </template>

          <v-list-item-title class="text-body-2">
            {{ item.title }}
            <v-badge
              v-if="item.badge"
              :content="item.badge"
              color="error"
              inline
              class="ml-2"
            ></v-badge>
          </v-list-item-title>
        </v-list-item>
      </v-list>
    </v-navigation-drawer>

    <!-- 主内容 -->
    <v-main>
      <v-container fluid class="pa-4">
        <!-- 发现视图 -->
        <div v-show="activeKey === 'discover'">
          <div v-if="peers.length > 0" class="peer-grid">
            <div v-for="peer in peers" :key="peer.id">
              <PeerCard
                :peer="peer"
                @transferStarted="activeKey = 'transfers'"
              />
            </div>
          </div>

          <div
            v-else
            class="empty-state d-flex flex-column justify-center align-center"
          >
            <v-icon
              icon="mdi-radar"
              size="100"
              color="primary"
              class="mb-4 radar-icon"
              style="opacity: 0.5"
            ></v-icon>
            <div class="text-grey">{{ t("discover.scanning") }}</div>
          </div>
        </div>

        <!-- 传输视图 -->
        <div v-show="activeKey === 'transfers'">
          <div v-if="transferList.length > 0">
            <div class="d-flex justify-end mb-2">
              <v-btn
                prepend-icon="mdi-delete-sweep"
                variant="text"
                color="error"
                @click="handleCleanFinished"
              >
                {{ t("transfers.clearFinished") }}
              </v-btn>
            </div>
            <TransferItem
              v-for="transfer in transferList"
              :key="transfer.id"
              :transfer="transfer"
            />
          </div>
          <div
            v-else
            class="empty-state d-flex flex-column justify-center align-center"
          >
            <v-icon icon="mdi-inbox" size="100" class="mb-4 text-grey"></v-icon>
            <div class="text-grey">{{ t("transfers.noTransfers") }}</div>
          </div>
        </div>

        <!-- 设置视图 -->
        <div v-show="activeKey === 'settings'">
          <SettingsView />
        </div>
      </v-container>
    </v-main>
  </v-layout>
</template>

<style scoped>
.empty-state {
  height: 80vh;
}

.radar-icon {
  animation: spin 3s linear infinite;
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

@media (min-width: 960px) {
  .peer-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}
</style>
