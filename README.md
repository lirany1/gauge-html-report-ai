# Gauge Enhanced HTML Report with AI Analytics````markdown

# Enhanced Gauge HTML Report

![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)

![Go Version](https://img.shields.io/badge/go-1.21+-blue.svg)> ✅ **STATUS: FULLY WORKING!** - Successfully generating HTML reports for Gauge test automation framework.  

![Status](https://img.shields.io/badge/status-active-green.svg)> 🎉 **Test Results:** 3 specs, 12 scenarios - all passing! | Report generated at `reports/html-report/index.html`

![CI](https://github.com/lirany1/gauge-html-report-ai/workflows/CI/badge.svg)

![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)

An enhanced HTML report generation plugin for the [Gauge](https://gauge.org) test automation framework with **AI-powered failure analysis**, advanced analytics, and modern UI.![Go Version](https://img.shields.io/badge/go-1.21+-blue.svg)

![Status](https://img.shields.io/badge/status-active-green.svg)

## 🚀 Live Demo

An enhanced HTML report generation plugin for the [Gauge](https://gauge.org) test automation framework with advanced analytics, modern UI, and extensive customization options.

**[View Interactive Demo →](https://lirany1.github.io/gauge-html-report-ai/)**

## ✨ Enhanced Features

## ✨ Key Features

### 🚀 **What's New Compared to Original**

### 🧠 **AI-Powered Analysis**- **📊 Advanced Analytics**: Interactive charts, trend analysis, flaky test detection

- **Intelligent Error Classification**: Automatically categorizes failures (Assertion, Timeout, Network, etc.)- **🎨 Modern UI**: Responsive design, dark/light themes, mobile support

- **Smart Failure Grouping**: Groups similar errors for focused debugging- **🔍 Smart Filtering**: Advanced search, tag-based filtering, custom presets

- **AI Insights**: Optional integration with OpenAI, Claude, or local LLMs for contextual fix suggestions- **📤 Export Options**: PDF reports, email sharing, Slack/Teams integration

- **Executive Summary**: Business-level health status and actionable recommendations- **⚡ Performance**: Real-time updates, optimized rendering for large test suites

- **🔧 Extensible**: Plugin architecture for custom visualizations and integrations

### 📊 **Advanced Analytics**

- **Interactive Dashboards**: Real-time charts and metrics### 📈 **Analytics Dashboard**

- **Trend Analysis**: Historical data comparison and regression detection- **Execution Timeline**: Visual timeline of test runs with performance metrics

- **Flaky Test Detection**: Automatically identify unstable tests- **Trend Analysis**: Historical data comparison and regression detection  

- **Performance Metrics**: Execution time analysis and bottleneck identification- **Flaky Test Detection**: Automatically identify unstable tests

- **Performance Metrics**: Detailed execution time analysis and bottleneck identification

### 🎨 **Modern UI/UX**- **Coverage Insights**: Test coverage visualization with drill-down capabilities

- **Responsive Design**: Optimized for desktop, tablet, and mobile

- **Dark/Light Themes**: User preference support### � **Intelligent Analysis & Optional AI Integration** ✨ NEW

- **Advanced Filtering**: Smart search, tag-based filtering, custom presets- **Dual-Mode System**: 

- **Export Options**: PDF reports, email sharing, integrations  - **Pattern-Based Intelligent Analysis** (Default): Fast, free, offline error classification

  - **AI-Powered Insights** (Optional): Context-aware analysis with OpenAI, Claude, or Local LLMs

## 🚀 Quick Start- **Automatic Error Classification**: 8 error types (Assertion, Timeout, Network, etc.)

- **Failure Grouping**: Smart clustering of similar failures with MD5 signatures

### Installation- **Executive Summary**: Business-level health status and recommendations

- **Fix Suggestions**: 

```bash  - Pattern-based: Template recommendations (works offline)

# Install from source  - AI-powered: Context-specific, code-aware solutions (requires API key)

git clone https://github.com/lirany1/gauge-html-report-ai.git- **Optional AI Providers**:

cd gauge-html-report-ai  - 🚀 **OpenAI** (GPT-4, GPT-3.5) - Best quality

go build -o gauge-html-report-enhanced ./cmd/html-report-enhanced  - 🧠 **Anthropic Claude** (Claude 3) - Detailed analysis

```  - 🏠 **Local LLM** (Ollama, LM Studio) - Privacy-preserving



### Basic Usage📚 **[AI Setup Guide](docs/REAL_AI_SETUP.md)** | **[How Analysis Works](docs/AI_ANALYSIS_EXPLAINED.md)**



```bash### 🎯 **Advanced Filtering & Search**

# Generate enhanced HTML report- **Smart Search**: Full-text search across descriptions, error messages, and logs

gauge run specs/ --reporter=html-enhanced- **Multi-dimensional Filtering**: Filter by status, tags, duration, execution date

```- **Custom Filter Presets**: Save and share commonly used filter combinations

- **Regex Support**: Advanced pattern matching for power users

### AI Configuration (Optional)- **Real-time Results**: Instant filtering without page reloads



```bash### 📱 **Modern User Experience**

# OpenAI GPT-4 (Best quality)- **Responsive Design**: Optimized for desktop, tablet, and mobile devices

export GAUGE_AI_ENABLED=true- **Dark/Light Themes**: User preference support with system theme detection

export GAUGE_AI_PROVIDER=openai- **Interactive Elements**: Collapsible sections, tooltips, and drill-down navigation

export GAUGE_AI_API_KEY=sk-your-api-key-here- **Accessibility**: WCAG 2.1 AA compliant for inclusive usage

- **Performance**: Lazy loading and virtual scrolling for large datasets

# Anthropic Claude (Detailed analysis)

export GAUGE_AI_ENABLED=true## 🚀 Quick Start

export GAUGE_AI_PROVIDER=claude

export GAUGE_AI_API_KEY=sk-ant-your-api-key-here### Prerequisites

- [Go](https://golang.org/) 1.21 or higher

# Local LLM (Privacy-preserving, free)- [Gauge](https://gauge.org) framework installed

export GAUGE_AI_ENABLED=true

export GAUGE_AI_PROVIDER=local### Installation

export GAUGE_AI_MODEL=llama2

``````bash

# Install from release

**Note:** AI is optional! Intelligent pattern-based analysis works out-of-the-box with no setup required.gauge install html-report-enhanced --file html-report-enhanced-1.0.0-linux.x86_64.zip



## 📁 Project Structure# Or build from source

git clone https://github.com/your-org/gauge-html-report-enhanced.git

```cd gauge-html-report-enhanced

gauge-html-report-ai/make install

├── cmd/                    # CLI entry points```

├── pkg/

│   ├── ai/                # AI analysis engine### Configuration

│   ├── analytics/         # Analytics and metrics

│   ├── builder/           # Report builderAdd to your `env/default.properties`:

│   ├── models/            # Data models

│   └── themes/            # Theme management```properties

├── web/themes/            # HTML templates and assets# Enhanced HTML Report Configuration

├── demo/                  # Demo reportsgauge_reports_dir=reports

└── .github/workflows/     # CI/CD pipelinesgauge_html_theme=enhanced-default

```gauge_enable_analytics=true

gauge_enable_trends=true

## 🧪 Testinggauge_max_screenshot_size=2MB

gauge_export_formats=html,pdf

```bashgauge_notification_channels=slack,email

# Run all tests```

go test ./...

### AI Configuration (Optional)

# Run tests with coverage

go test -v -race -coverprofile=coverage.out ./...Enable AI-powered insights for advanced analysis (optional - intelligent analysis works without this):



# View coverage report```bash

go tool cover -html=coverage.out# Option 1: OpenAI (GPT-4) - Best Quality

```export GAUGE_AI_ENABLED=true

export GAUGE_AI_PROVIDER=openai

## 🛠 Developmentexport GAUGE_AI_API_KEY=sk-your-api-key-here

export GAUGE_AI_MODEL=gpt-4-turbo-preview  # or gpt-3.5-turbo

### Prerequisites

- Go 1.21 or higher# Option 2: Anthropic Claude - Detailed Analysis

- [Gauge](https://gauge.org) framework installedexport GAUGE_AI_ENABLED=true

export GAUGE_AI_PROVIDER=claude

### Buildingexport GAUGE_AI_API_KEY=sk-ant-your-api-key-here



```bash# Option 3: Local LLM (Ollama) - Privacy-Preserving, Free

# Download dependenciesexport GAUGE_AI_ENABLED=true

go mod downloadexport GAUGE_AI_PROVIDER=local

export GAUGE_AI_MODEL=llama2

# Build binary```

go build -o gauge-html-report-enhanced ./cmd/html-report-enhanced

**Note:** AI is optional! Pattern-based intelligent analysis works out-of-the-box with no setup.

# Run tests

go test ./...📖 **[Complete AI Setup Guide](docs/REAL_AI_SETUP.md)** - Detailed instructions, costs, and comparison

```

## 📊 Usage Examples

## 🤝 Contributing

### Basic Report Generation

1. Fork the repository```bash

2. Create a feature branch: `git checkout -b feature/amazing-feature`gauge run specs/ --reporter=html-enhanced

3. Make your changes and add tests```

4. Run the test suite: `go test ./...`

5. Submit a pull request### Advanced Configuration

```bash

## 📝 License# Generate with custom theme and analytics

gauge run specs/ \

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.  --reporter=html-enhanced \

  --theme=dark \

## 🙏 Acknowledgments  --enable-trends \

  --export-pdf

- Original [Gauge HTML Report](https://github.com/getgauge/html-report) by ThoughtWorks```

- [Gauge Framework](https://gauge.org) team

- Open source community### Programmatic API

```go

---import "github.com/your-org/gauge-html-report-enhanced/generator"



**Gauge Enhanced HTML Report** - AI-powered test reporting for modern development teams! 🚀config := &generator.Config{
    ThemePath: "themes/custom",
    EnableAnalytics: true,
    ExportFormats: []string{"html", "pdf"},
}

err := generator.GenerateEnhancedReport(suiteResult, config)
```

## 🎨 Themes & Customization

### Built-in Themes
- **Enhanced Default**: Modern, clean design with advanced features
- **Dark Mode**: Dark theme optimized for low-light environments
- **Classic**: Traditional layout with enhanced functionality
- **Minimal**: Clean, distraction-free reporting
- **Corporate**: Professional theme for enterprise environments

### Custom Themes
Create your own themes by extending our base templates:

```bash
gauge-html-enhanced create-theme --name="my-theme" --base="enhanced-default"
```

## 🔧 Advanced Features

### Plugin Architecture
Extend functionality with custom plugins:

```javascript
// plugins/custom-analytics.js
class CustomAnalyticsPlugin {
  processResults(suiteResult) {
    // Custom analytics logic
    return enhancedData;
  }
  
  renderWidget(data) {
    // Custom visualization
    return htmlWidget;
  }
}
```

### Integration Hooks
```yaml
# .gauge-enhanced.yml
integrations:
  slack:
    webhook_url: "${SLACK_WEBHOOK}"
    channel: "#test-results"
    template: "detailed"
  
  jenkins:
    job_url: "${BUILD_URL}"
    build_number: "${BUILD_NUMBER}"
    
  jira:
    project_key: "TEST"
    auto_create_issues: true
```

## 📤 Export & Sharing

### PDF Reports
```bash
# Generate PDF alongside HTML
gauge run specs/ --export-pdf --pdf-template="executive-summary"
```

### Email Integration
```bash
# Automatically email results
gauge run specs/ --email-results --recipients="team@company.com"
```

### CI/CD Integration
```yaml
# GitHub Actions example
- name: Run Tests & Generate Enhanced Report
  uses: gauge-org/gauge-html-enhanced-action@v1
  with:
    specs: 'specs/'
    theme: 'corporate'
    export-formats: 'html,pdf'
    slack-webhook: ${{ secrets.SLACK_WEBHOOK }}
```

## 🛠 Development

### Building from Source
```bash
git clone https://github.com/your-org/gauge-html-report-enhanced.git
cd gauge-html-report-enhanced

# Install dependencies
go mod download
npm install

# Build
make build

# Run tests
make test

# Create distribution
make dist
```

### Project Structure
```
gauge-html-report-enhanced/
├── cmd/                    # CLI entry points
├── pkg/
│   ├── analytics/         # Analytics engine
│   ├── generator/         # Report generation
│   ├── themes/           # Theme management
│   ├── export/           # Export functionality
│   └── integrations/     # Third-party integrations
├── web/
│   ├── src/              # Frontend source
│   ├── themes/           # Theme templates
│   └── dist/             # Built assets
├── examples/             # Usage examples
├── docs/                 # Documentation
└── scripts/              # Build scripts
```

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

### Development Setup
1. Fork the repository
2. Create a feature branch: `git checkout -b feature/amazing-feature`
3. Make your changes and add tests
4. Run the test suite: `make test`
5. Submit a pull request

## 📝 License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Original [Gauge HTML Report](https://github.com/getgauge/html-report) by ThoughtWorks
- [Gauge Framework](https://gauge.org) team
- Open source contributors and community

## 📞 Support

- 📖 [Documentation](https://docs.gauge-enhanced-html-report.org)
- 💬 [Community Discussions](https://github.com/your-org/gauge-html-report-enhanced/discussions)  
- 🐛 [Issue Tracker](https://github.com/your-org/gauge-html-report-enhanced/issues)
- 📧 [Email Support](mailto:support@gauge-enhanced-html-report.org)

---

**Enhanced Gauge HTML Report** - Taking test reporting to the next level! 🚀