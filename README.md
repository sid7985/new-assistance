# New Assistance (formerly NanoClaw)

<p align="center">
  <img src="assets/nanoclaw-logo.png" alt="New Assistance" width="400">
</p>

<p align="center">
  An AI assistant that runs agents securely in their own containers. Lightweight, built to be easily understood and completely customized for your needs.
</p>

<p align="center">
  <a href="https://nanoclaw.dev">nanoclaw.dev</a>&nbsp; • &nbsp;
  <a href="https://docs.nanoclaw.dev">docs</a>&nbsp; • &nbsp;
  <a href="README_zh.md">中文</a>&nbsp; • &nbsp;
  <a href="README_ja.md">日本語</a>&nbsp; • &nbsp;
  <a href="https://discord.gg/VDdww8qS42"><img src="https://img.shields.io/discord/1470188214710046894?label=Discord&logo=discord&v=2" alt="Discord" valign="middle"></a>
</p>

---

## 🚀 Overview

**New Assistance** is a secure, lightweight AI agentic system designed for individuals who want full control and isolation for their AI assistants. Built on the core of NanoClaw, it focuses on security through OS-level isolation (Docker/Apple Container) rather than just application-level checks.

## ✨ Key Features

- **🛡️ Secure by Isolation** - Agents run in their own Linux containers (or Apple Containers). Bash access is safe because it's restricted to the sandbox.
- **📱 Multi-Channel Messaging** - Use WhatsApp, Telegram, Discord, Slack, or Gmail to interact with your assistant.
- **🧠 Isolated Group Context** - Each group has its own `CLAUDE.md` memory and filesystem, ensuring no leakage between separate tasks or conversations.
- **⚙️ AI-Native Workflow** - No complex setup wizards or dashboards. Everything is managed through the Claude CLI.
- **🏗️ Skill-Based Extensibility** - Add new features like `/add-whatsapp` or `/add-gmail` as skills that modify the codebase directly, keeping it lean.

## 🛠️ NanoClaw Orchestrator

This repository now includes the consolidated **NanoClaw Orchestrator**, a Go-based runtime for powering advanced agentic behaviors:

- **🤖 Diverse Model Support**: Integration with MiniMax, Perplexity, and Ollama.
- **🌐 Web Core**: Built-in Selenium and WebDriver support for automated web tasks.
- **📦 Container Automation**: Enhanced management of isolated Docker/MicroVM sandboxes.
- **📺 Media Control**: Scripts for YouTube playback and media management.

## 🛠️ Quick Start

```bash
git clone https://github.com/sid7985/new-assistance.git
cd new-assistance
claude
```

Then run `/setup` inside the `claude` prompt. Claude Code handles the dependencies, authentication, and container setup.

## 🏛️ Architecture

```
Channels --> SQLite --> Polling loop --> Container (Claude Agent SDK) --> Response
```

- **One Process**: A single Node.js process manages everything.
- **SQLite**: Stores messages, groups, and state.
- **Isolated Containers**: Spawns streaming agent containers for each session.
- **Agent Vault**: Credentials are never stored inside the agent container, adding an extra layer of security.

## 📄 License

This project is licensed under the MIT License.
