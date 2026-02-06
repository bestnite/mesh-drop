# Mesh Drop

简易、快速的局域网文件传输工具，基于 Wails 和 Vue 构建。

## 功能特性

- **文件传输**：支持多文件发送，轻松共享。
- **文件夹传输**：支持发送整个文件夹结构。
- **文本传输**：快速同步设备间的文本内容。
- **加密传输**：确保数据在传输过程中的安全性。
- **安全身份**：基于 Ed25519 的签名验证，防止伪造。

## 安全机制

Mesh Drop 采用多层安全设计来保护用户免受潜在的恶意攻击：

1.  **身份验证 (Identity)**
    - 每个设备在首次启动时生成一对唯一的 Ed25519 密钥。
    - 所有广播包（Presence Broadcast）都使用私钥签名。
    - 接收端通过公钥验证签名，确保身份未被篡改。

2.  **信任机制 (Trust)**
    - 采用 TOFU (Trust On First Use) 策略。
    - 用户可以选择“信任”某个 Peer，一旦信任，该 Peer 的公钥将被固定（Pinning）。
    - 之后收到该 Peer ID 的所有数据包，必须通过已保存公钥的验证，否则会被标记为 **Mismatch**。
    - **防欺骗**：如果有人试图伪造已信任 Peer 的 ID，UI 会显示明显的“Mismatch”安全警告，并阻止元数据被覆盖。

3.  **传输加密 (Encryption)**
    - 文件传输服务使用 HTTPS 协议。
    - 自动生成自签名证书进行通信加密，防止传输内容被窃听。

## 截图

| ![Mesh Drop](./screenshot/1.png) | ![Mesh Drop](./screenshot/2.png) |
| -------------------------------- | -------------------------------- |

## 待办事项

- [x] 剪辑板传输
- [x] 文件夹传输
- [x] 取消传输
- [x] 多文件发送
- [x] 加密传输
- [x] 设置页面
- [x] 单例模式
- [x] 系统通知
- [x] 清理历史
- [x] 自动接收
- [x] 应用图标
- [x] 信任Peer
- [ ] 系统托盘（最小化到托盘）徽章 https://github.com/wailsapp/wails/issues/4494
- [ ] 多语言

## 技术栈

本项目使用现代化的技术栈构建：

- **后端**: [Go](https://go.dev/) + [Wails v3](https://v3.wails.io/)
- **前端**: [Vue 3](https://vuejs.org/) + [TypeScript](https://www.typescriptlang.org/)
- **UI 框架**: [Vuetify](https://vuetifyjs.com/)

## 开发

### 前置条件

在开始之前，请确保您的开发环境已安装以下工具：

1.  **Go** (版本 >= 1.25)
2.  **Node.js**
3.  **Wails CLI**
4.  **UPX**

### 安装依赖

```bash
# 进入项目目录
cd mesh-drop

# 安装前端依赖 (通常 Wails 会自动处理，但手动安装可确保环境清晰)
cd frontend
npm install
cd ..
```

### 运行开发环境

```bash
wails3 dev
```
