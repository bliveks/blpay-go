(() => {
  let lastOpener = null;

  const toast = document.querySelector('.toast');
  if (toast) {
    window.setTimeout(() => {
      toast.classList.add('toast-hidden');
      window.setTimeout(() => toast.remove(), 250);
    }, 5000);
  }

  document.addEventListener('click', (event) => {
    const openButton = event.target.closest('[data-modal-open]');
    if (openButton) {
      const dialog = document.getElementById(openButton.dataset.modalOpen);
      if (dialog instanceof HTMLDialogElement) {
        lastOpener = openButton;
        dialog.showModal();
        dialog.querySelector('input:not([type="hidden"]), select, textarea')?.focus();
      }
      return;
    }

    const closeButton = event.target.closest('[data-modal-close]');
    if (closeButton) {
      closeButton.closest('dialog')?.close();
      return;
    }

    if (event.target instanceof HTMLDialogElement && event.target.open) {
      const bounds = event.target.getBoundingClientRect();
      const outside = event.clientX < bounds.left || event.clientX > bounds.right || event.clientY < bounds.top || event.clientY > bounds.bottom;
      if (outside) event.target.close();
    }
  });

  document.querySelectorAll('dialog.modal').forEach((dialog) => {
    dialog.addEventListener('close', () => lastOpener?.focus());
  });

  const sidebar = document.getElementById('sidebar');
  const menuToggle = document.getElementById('menu-toggle');
  const closeMenu = () => {
    sidebar?.classList.remove('open');
    menuToggle?.setAttribute('aria-expanded', 'false');
  };

  menuToggle?.addEventListener('click', () => {
    const isOpen = sidebar?.classList.toggle('open') ?? false;
    menuToggle.setAttribute('aria-expanded', String(isOpen));
  });

  document.addEventListener('click', (event) => {
    if (sidebar?.classList.contains('open') && !sidebar.contains(event.target) && !menuToggle?.contains(event.target)) closeMenu();
  });

  document.addEventListener('keydown', (event) => {
    if (event.key === 'Escape' && !document.querySelector('dialog[open]')) closeMenu();
  });

  document.querySelectorAll('[data-category-form]').forEach((form) => {
    const type = form.querySelector('select[name="type"]');
    const limitField = form.querySelector('[data-expense-limit]');
    const limitInput = form.querySelector('input[name="limit"]');
    if (!type || !limitField || !limitInput) return;

    const syncLimitField = () => {
      const isExpense = type.value === 'xerc';
      limitField.hidden = !isExpense;
      limitInput.disabled = !isExpense;
    };

    type.addEventListener('change', syncLimitField);
    syncLimitField();
  });
})();
