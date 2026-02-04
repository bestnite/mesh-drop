<script lang="ts" setup>
import { onMounted, ref, computed } from "vue";
import PeerCard from "./PeerCard.vue";
import TransferItem from "./TransferItem.vue";
import { Peer } from "../../bindings/mesh-drop/internal/discovery/models";
import { Transfer } from "../../bindings/mesh-drop/internal/transfer";
import { GetPeers } from "../../bindings/mesh-drop/internal/discovery/service";
import { Events } from "@wailsio/runtime";
import { GetTransferList } from "../../bindings/mesh-drop/internal/transfer/service";
import {
  GetSavePath,
  SetSavePath,
  GetHostName,
  SetHostName,
  GetAutoAccept,
  SetAutoAccept,
  GetSaveHistory,
  SetSaveHistory,
} from "../../bindings/mesh-drop/internal/config/config";
import { Dialogs } from "@wailsio/runtime";

const peers = ref<Peer[]>([]);
const transferList = ref<Transfer[]>([]);
const activeKey = ref("discover");
const drawer = ref(true);
const isMobile = ref(false);

// 监听窗口大小变化更新 isMobile
onMounted(async () => {
  checkMobile();
  window.addEventListener("resize", checkMobile);
  const list = await GetTransferList();
  transferList.value = (
    (list || []).filter((t) => t !== null) as Transfer[]
  ).sort((a, b) => b.create_time - a.create_time);

  if (isMobile.value) {
    drawer.value = false;
  }

  // 加载配置
  savePath.value = await GetSavePath();
  hostName.value = await GetHostName();
  autoAccept.value = await GetAutoAccept();
  saveHistory.value = await GetSaveHistory();
});

const checkMobile = () => {
  const mobile = window.innerWidth < 768;
  if (mobile !== isMobile.value) {
    isMobile.value = mobile;
    drawer.value = !mobile;
  }
};

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

const menuItems = computed(() => [
  {
    title: "Discover",
    value: "discover",
    icon: "mdi-radar",
  },
  {
    title: "Transfers",
    value: "transfers",
    icon: "mdi-inbox",
    badge: pendingCount.value > 0 ? pendingCount.value : null,
  },
  {
    title: "Settings",
    value: "settings",
    icon: "mdi-cog",
  },
]);

// --- 设置 ---
const savePath = ref("");

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

const hostName = ref("");
const autoAccept = ref(false);
const saveHistory = ref(false);

// --- 操作 ---

const handleMenuClick = (key: string) => {
  activeKey.value = key;
  if (isMobile.value) {
    drawer.value = false;
  }
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
      <div class="pa-4" v-if="!isMobile">
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

          <v-list-item-title>
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
            <div class="text-grey">Scanning for peers...</div>
          </div>
        </div>

        <!-- 传输视图 -->
        <div v-show="activeKey === 'transfers'">
          <div v-if="transferList.length > 0">
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
            <div class="text-grey">No transfers yet</div>
          </div>
        </div>

        <!-- 设置视图 -->
        <div v-show="activeKey === 'settings'">
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
              <template #append
                ><v-text-field
                  clearable
                  variant="underlined"
                  v-model="hostName"
                  width="200"
                  @update:modelValue="SetHostName"
                ></v-text-field
              ></template>
            </v-list-item>
            <v-list-item title="Save History">
              <template #prepend>
                <v-icon icon="mdi-history"></v-icon>
              </template>
              <template #append
                ><v-switch
                  v-model="saveHistory"
                  color="primary"
                  inset
                  hide-details
                  @update:modelValue="SetSaveHistory(saveHistory)"
                ></v-switch
              ></template>
            </v-list-item>
            <v-list-item title="Auto Accept">
              <template #prepend>
                <v-icon icon="mdi-content-save"></v-icon>
              </template>
              <template #append
                ><v-switch
                  v-model="autoAccept"
                  color="primary"
                  inset
                  hide-details
                  @update:modelValue="SetAutoAccept(autoAccept)"
                ></v-switch
              ></template>
            </v-list-item>
          </v-list>
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
