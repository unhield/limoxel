// Limoxel SDK Developer Portal Client Logic

document.addEventListener('DOMContentLoaded', () => {
  // Initialize Copy-to-Clipboard buttons
  document.querySelectorAll('.copy-btn').forEach(btn => {
    btn.addEventListener('click', () => {
      const codeBlock = btn.closest('.code-box').querySelector('code');
      if (codeBlock) {
        navigator.clipboard.writeText(codeBlock.innerText.trim()).then(() => {
          const orig = btn.innerText;
          btn.innerText = 'Copied!';
          setTimeout(() => { btn.innerText = orig; }, 2000);
        });
      }
    });
  });

  // Filter functionality for API Explorer and Examples
  const searchInput = document.getElementById('search-filter');
  if (searchInput) {
    searchInput.addEventListener('input', (e) => {
      const q = e.target.value.toLowerCase().trim();
      document.querySelectorAll('.filterable-item').forEach(item => {
        const text = item.innerText.toLowerCase();
        if (text.includes(q)) {
          item.style.display = '';
        } else {
          item.style.display = 'none';
        }
      });
    });
  }
});
