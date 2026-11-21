(() => {
  let lastOpener = null;

  const validMonth = (value) => /^\d{4}-(0[1-9]|1[0-2])$/.test(value);
  const saveMonth = (value) => {
    if (!validMonth(value)) return;
    const secure = window.location.protocol === 'https:' ? '; Secure' : '';
    document.cookie = `blpay_month=${value}; Path=/; SameSite=Lax${secure}`;
  };
  const navigateWithoutMonthQuery = () => {
    const url = new URL(window.location.href);
    url.searchParams.delete('month');
    window.location.assign(`${url.pathname}${url.search}${url.hash}`);
  };

  const currentURL = new URL(window.location.href);
  const queryMonth = currentURL.searchParams.get('month');
  if (validMonth(queryMonth || '')) saveMonth(queryMonth);
  if (currentURL.searchParams.has('month')) {
    currentURL.searchParams.delete('month');
    window.history.replaceState({}, '', `${currentURL.pathname}${currentURL.search}${currentURL.hash}`);
  }

  document.querySelectorAll('[data-month-select]').forEach((select) => {
    select.addEventListener('change', () => {
      saveMonth(select.value);
      navigateWithoutMonthQuery();
    });
  });

  document.querySelectorAll('[data-month-form]').forEach((form) => {
    form.addEventListener('submit', (event) => {
      event.preventDefault();
      saveMonth(form.elements.namedItem('month').value);
      navigateWithoutMonthQuery();
    });
  });

  const toast = document.querySelector('.toast');
  if (toast) {
    window.setTimeout(() => {
      toast.classList.add('toast-hidden');
      window.setTimeout(() => toast.remove(), 250);
    }, 5000);
  }

  document.addEventListener('click', (event) => {
    const categoryEditButton = event.target.closest('[data-category-edit]');
    if (categoryEditButton) {
      const form = document.querySelector('#category-edit-modal form');
      if (form) {
        form.elements.namedItem('id').value = categoryEditButton.dataset.id;
        form.elements.namedItem('name').value = categoryEditButton.dataset.name;
        form.elements.namedItem('type').value = categoryEditButton.dataset.type;
        form.elements.namedItem('limit').value = categoryEditButton.dataset.limit;
        form.elements.namedItem('type').dispatchEvent(new Event('change'));
      }
    }

    const editButton = event.target.closest('[data-transaction-edit]');
    if (editButton) {
      const form = document.querySelector('#transaction-edit-modal form');
      if (form) {
        form.elements.namedItem('id').value = editButton.dataset.id;
        form.elements.namedItem('amount').value = editButton.dataset.amount;
        form.elements.namedItem('date').value = editButton.dataset.date;
        form.elements.namedItem('account').value = editButton.dataset.account;
        form.elements.namedItem('note').value = editButton.dataset.note;
        const category = form.elements.namedItem('category');
        const option = Array.from(category.options).find((item) => item.textContent === editButton.dataset.category);
        if (option) category.value = option.value;
      }
    }

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

  document.querySelectorAll('[data-category-type-filter]').forEach((filter) => {
    filter.addEventListener('change', () => {
      document.querySelectorAll('[data-category-type]').forEach((row) => {
        row.hidden = Boolean(filter.value) && row.dataset.categoryType !== filter.value;
      });
    });
  });
})();
