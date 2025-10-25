// Enhanced Gauge Report - Main JavaScript

class EnhancedReport {
  constructor() {
    this.currentTheme = localStorage.getItem('theme') || 'light';
    this.currentTab = 'overview';
    this.init();
  }

  init() {
    this.setupTheme();
    this.setupNavigation();
    this.setupFilters();
    this.setupCharts();
    this.setupExport();
    this.setupSearch();
  }

  // Theme Management
  setupTheme() {
    const themeToggle = document.getElementById('themeToggle');
    this.applyTheme(this.currentTheme);

    themeToggle?.addEventListener('click', () => {
      this.currentTheme = this.currentTheme === 'light' ? 'dark' : 'light';
      this.applyTheme(this.currentTheme);
      localStorage.setItem('theme', this.currentTheme);
    });
  }

  applyTheme(theme) {
    document.body.setAttribute('data-theme', theme);
    document.body.className = `theme-${theme}`;
  }

  // Navigation
  setupNavigation() {
    const navItems = document.querySelectorAll('.nav__item');
    
    navItems.forEach(item => {
      item.addEventListener('click', (e) => {
        const tab = e.currentTarget.getAttribute('data-tab');
        this.switchTab(tab);
      });
    });
  }

  switchTab(tabName) {
    // Hide all tabs
    document.querySelectorAll('.tab-content').forEach(tab => {
      tab.classList.remove('active');
    });

    // Show selected tab
    const selectedTab = document.getElementById(tabName);
    if (selectedTab) {
      selectedTab.classList.add('active');
    }

    // Update nav items
    document.querySelectorAll('.nav__item').forEach(item => {
      item.classList.remove('active');
      if (item.getAttribute('data-tab') === tabName) {
        item.classList.add('active');
      }
    });

    this.currentTab = tabName;
  }

  // Filters
  setupFilters() {
    const searchInput = document.getElementById('specSearch');
    const statusFilter = document.getElementById('statusFilter');

    searchInput?.addEventListener('input', (e) => {
      this.filterSpecs(e.target.value, statusFilter?.value || 'all');
    });

    statusFilter?.addEventListener('change', (e) => {
      this.filterSpecs(searchInput?.value || '', e.target.value);
    });
  }

  filterSpecs(searchTerm, status) {
    const specCards = document.querySelectorAll('.spec-card');
    searchTerm = searchTerm.toLowerCase();

    specCards.forEach(card => {
      const title = card.querySelector('.spec-card__title')?.textContent.toLowerCase() || '';
      const cardStatus = card.classList.contains('spec-card--passed') ? 'passed' :
                         card.classList.contains('spec-card--failed') ? 'failed' :
                         card.classList.contains('spec-card--skipped') ? 'skipped' : '';

      const matchesSearch = title.includes(searchTerm);
      const matchesStatus = status === 'all' || cardStatus === status;

      card.style.display = (matchesSearch && matchesStatus) ? 'block' : 'none';
    });
  }

  // Charts
  setupCharts() {
    this.setupResultsChart();
    this.setupSuccessRateCircle();
  }

  setupResultsChart() {
    const canvas = document.getElementById('resultsChart');
    if (!canvas) return;

    const ctx = canvas.getContext('2d');
    const data = this.getTestResultsData();

    new Chart(ctx, {
      type: 'doughnut',
      data: {
        labels: ['Passed', 'Failed', 'Skipped'],
        datasets: [{
          data: [data.passed, data.failed, data.skipped],
          backgroundColor: [
            'rgba(16, 185, 129, 0.8)',
            'rgba(239, 68, 68, 0.8)',
            'rgba(245, 158, 11, 0.8)'
          ],
          borderColor: [
            'rgba(16, 185, 129, 1)',
            'rgba(239, 68, 68, 1)',
            'rgba(245, 158, 11, 1)'
          ],
          borderWidth: 2
        }]
      },
      options: {
        responsive: true,
        maintainAspectRatio: true,
        plugins: {
          legend: {
            position: 'bottom',
          },
          tooltip: {
            callbacks: {
              label: function(context) {
                const label = context.label || '';
                const value = context.parsed || 0;
                const total = context.dataset.data.reduce((a, b) => a + b, 0);
                const percentage = Math.round((value / total) * 100);
                return `${label}: ${value} (${percentage}%)`;
              }
            }
          }
        }
      }
    });
  }

  setupSuccessRateCircle() {
    const circle = document.querySelector('.success-rate__circle');
    if (!circle) return;

    const successRate = parseFloat(circle.closest('.success-rate').dataset.rate || 0);
    circle.style.setProperty('--success-rate', successRate);
  }

  getTestResultsData() {
    // Extract data from summary cards
    const summaryCards = document.querySelectorAll('.summary-card');
    let passed = 0, failed = 0, skipped = 0;

    summaryCards.forEach(card => {
      const value = parseInt(card.querySelector('.summary-card__value')?.textContent || 0);
      if (card.classList.contains('summary-card--success')) {
        passed = value;
      } else if (card.classList.contains('summary-card--danger')) {
        failed = value;
      } else if (card.classList.contains('summary-card--warning')) {
        skipped = value;
      }
    });

    return { passed, failed, skipped };
  }

  // Export functionality
  setupExport() {
    const exportBtn = document.getElementById('exportBtn');
    exportBtn?.addEventListener('click', () => {
      this.showExportMenu();
    });
  }

  showExportMenu() {
    // Show export options
    const options = ['PDF', 'JSON', 'CSV', 'XML'];
    const menu = document.createElement('div');
    menu.className = 'export-menu';
    menu.innerHTML = `
      <div class="export-menu__content">
        <h3>Export Report</h3>
        ${options.map(opt => `
          <button class="export-option" data-format="${opt.toLowerCase()}">
            Export as ${opt}
          </button>
        `).join('')}
      </div>
    `;

    document.body.appendChild(menu);

    menu.querySelectorAll('.export-option').forEach(btn => {
      btn.addEventListener('click', (e) => {
        const format = e.target.getAttribute('data-format');
        this.exportReport(format);
        menu.remove();
      });
    });

    // Close on outside click
    setTimeout(() => {
      document.addEventListener('click', (e) => {
        if (!menu.contains(e.target)) {
          menu.remove();
        }
      }, { once: true });
    }, 100);
  }

  exportReport(format) {
    // Get current report data
    const reportData = this.getReportData();
    
    switch(format) {
      case 'json':
        this.downloadJSON(reportData);
        break;
      case 'csv':
        this.downloadCSV(reportData);
        break;
      case 'pdf':
        this.generatePDF();
        break;
      case 'xml':
        this.downloadXML(reportData);
        break;
      default:
        console.warn('Unknown export format:', format);
    }
  }
  
  getReportData() {
    // Extract data from the current page
    const summaryCards = document.querySelectorAll('.summary-card');
    const specs = document.querySelectorAll('.spec-card');
    
    const summary = {};
    summaryCards.forEach(card => {
      const label = card.querySelector('.summary-card__label')?.textContent.toLowerCase();
      const value = card.querySelector('.summary-card__value')?.textContent;
      if (label && value) {
        summary[label] = value;
      }
    });
    
    const specifications = [];
    specs.forEach(spec => {
      const title = spec.querySelector('.spec-card__title')?.textContent;
      const status = spec.querySelector('.status-badge')?.textContent;
      const meta = spec.querySelector('.spec-card__meta')?.textContent;
      
      if (title) {
        specifications.push({ title, status, meta });
      }
    });
    
    return {
      summary,
      specifications,
      exportedAt: new Date().toISOString(),
      projectName: document.querySelector('h1')?.textContent || 'Test Report'
    };
  }
  
  downloadJSON(data) {
    const jsonStr = JSON.stringify(data, null, 2);
    this.downloadFile(jsonStr, 'report.json', 'application/json');
  }
  
  downloadCSV(data) {
    let csv = 'Specification,Status,Meta\n';
    data.specifications.forEach(spec => {
      csv += `"${spec.title}","${spec.status}","${spec.meta}"\n`;
    });
    this.downloadFile(csv, 'report.csv', 'text/csv');
  }
  
  downloadXML(data) {
    const xml = `<?xml version="1.0" encoding="UTF-8"?>
<testReport>
  <project>${data.projectName}</project>
  <exportedAt>${data.exportedAt}</exportedAt>
  <summary>
    <passed>${data.summary.passed || 0}</passed>
    <failed>${data.summary.failed || 0}</failed>
    <skipped>${data.summary.skipped || 0}</skipped>
  </summary>
  <specifications>
    ${data.specifications.map(spec => 
      `<specification>
        <title>${spec.title}</title>
        <status>${spec.status}</status>
        <meta>${spec.meta}</meta>
      </specification>`
    ).join('\n    ')}
  </specifications>
</testReport>`;
    this.downloadFile(xml, 'report.xml', 'application/xml');
  }
  
  generatePDF() {
    // Simple PDF generation using browser's print functionality
    const printWindow = window.open('', '_blank');
    const html = `
      <!DOCTYPE html>
      <html>
      <head>
        <title>Test Report</title>
        <style>
          body { font-family: Arial, sans-serif; margin: 20px; }
          .header { text-align: center; margin-bottom: 20px; }
          .summary { margin-bottom: 20px; }
          .specs { margin-top: 20px; }
          .spec { margin-bottom: 10px; padding: 10px; border: 1px solid #ddd; }
          .passed { background-color: #d4edda; }
          .failed { background-color: #f8d7da; }
          .skipped { background-color: #fff3cd; }
        </style>
      </head>
      <body>
        ${document.querySelector('main').innerHTML}
      </body>
      </html>
    `;
    
    printWindow.document.write(html);
    printWindow.document.close();
    printWindow.focus();
    printWindow.print();
    printWindow.close();
  }
  
  downloadFile(content, filename, contentType) {
    const blob = new Blob([content], { type: contentType });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = filename;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
  }

  // Search functionality
  setupSearch() {
    const searchInput = document.getElementById('specSearch');
    if (!searchInput) return;

    // Add debounce to improve performance
    let searchTimeout;
    searchInput.addEventListener('input', (e) => {
      clearTimeout(searchTimeout);
      searchTimeout = setTimeout(() => {
        this.performSearch(e.target.value);
      }, 300);
    });
  }

  performSearch(query) {
    if (!query) {
      this.filterSpecs('', 'all');
      return;
    }

    // Advanced search logic can be added here
    this.filterSpecs(query, document.getElementById('statusFilter')?.value || 'all');
  }
}

// Initialize the enhanced report when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
  window.enhancedReport = new EnhancedReport();
  console.log('Enhanced Gauge Report initialized');
});