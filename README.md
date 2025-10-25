# 🚀 Gauge Enhanced HTML Report with AI

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.21+-blue.svg)](https://golang.org/)
[![Version](https://img.shields.io/badge/version-5.0.0-brightgreen.svg)](https://github.com/lirany1/gauge-html-report-ai/releases)
[![CI](https://github.com/lirany1/gauge-html-report-ai/workflows/CI/badge.svg)](https://github.com/lirany1/gauge-html-report-ai/actions)

Transform your [Gauge](https://gauge.org) test reports with **AI-powered insights** and modern analytics! 

## ✨ What makes it special?

🧠 **Smart AI Analysis** - Automatically categorizes failures and suggests fixes  
📊 **Beautiful Reports** - Modern, responsive design with interactive charts  
🔍 **Intelligent Grouping** - Groups similar errors for faster debugging  
⚡ **Works Offline** - No AI setup required, works great out of the box!

## 🎮 Try the Demo

**[→ See Live Demo](https://lirany1.github.io/gauge-html-report-ai/)**

## ⚡ Quick Start

```bash
# 1. Clone and build
git clone https://github.com/lirany1/gauge-html-report-ai.git
cd gauge-html-report-ai
go build -o gauge-html-report-enhanced ./cmd/html-report-enhanced

# 2. Generate your first enhanced report
gauge run specs/ --reporter=html-enhanced
```

That's it! 🎉 Your enhanced report will be in `reports/html-report/index.html`

## 🔧 Optional: Add AI Superpowers

Want even smarter analysis? Add an AI provider:

```bash
# OpenAI (best quality)
export GAUGE_AI_ENABLED=true
export GAUGE_AI_PROVIDER=openai
export GAUGE_AI_API_KEY=your-openai-key

# Or use local LLM (free & private)
export GAUGE_AI_ENABLED=true
export GAUGE_AI_PROVIDER=local
export GAUGE_AI_MODEL=llama2
```

## 🎯 Key Features

| Feature | Description |
|---------|-------------|
| 🤖 **AI Error Analysis** | Automatically categorizes failures (Timeout, Network, etc.) |
| 📈 **Analytics Engine** | Performance metrics and execution timeline |
| 🔍 **Smart Grouping** | Groups similar failures together |
| 📱 **Mobile Friendly** | Responsive design works on all devices |
| ⚡ **Pattern Matching** | Intelligent analysis works without AI setup |
| 🧠 **Multi-LLM Support** | OpenAI, Claude, and local LLM providers |

## 🧪 Testing

```bash
go test ./...  # Run all tests
```

## 🤝 Contributing

Found a bug or have an idea? [Open an issue](https://github.com/lirany1/gauge-html-report-ai/issues) or submit a PR!

## 📄 License

Apache 2.0 - see [LICENSE](LICENSE)

---

Made with ❤️ for the testing community
