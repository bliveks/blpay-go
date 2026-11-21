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
      document.querySelectorAll('[data-category-type-filter]').forEach((otherFilter) => {
        otherFilter.value = filter.value;
      });
      document.querySelectorAll('[data-category-type]').forEach((row) => {
        row.hidden = Boolean(filter.value) && row.dataset.categoryType !== filter.value;
      });
    });
  });

  document.querySelectorAll('[data-category-filter-toggle]').forEach((button) => {
    button.addEventListener('click', () => {
      const filter = button.closest('[data-category-filter]')?.querySelector('[data-category-type-filter]');
      if (!filter) return;
      filter.hidden = !filter.hidden;
      if (!filter.hidden) filter.focus();
    });
  });

  const reportFilters = Array.from(document.querySelectorAll('[data-report-filter]'));
  const reportRows = Array.from(document.querySelectorAll('[data-report-row]'));
  if (reportFilters.length && reportRows.length) {
    const emptyRow = document.querySelector('.report-filter-empty');
    const amountFilter = reportFilters.find((filter) => filter.dataset.reportFilter === 'amount');
    const totalOutput = document.querySelector('[data-report-total]');
    const matchesFilter = (row, filter) => {
      const value = filter.value;
      if (!value) return true;
      const field = filter.dataset.reportFilter;
      if (field === 'amount') return Number(row.dataset.amount).toFixed(2) === value;
      if (field === 'note' && value === '__empty__') return !row.dataset.note;
      return row.dataset[field] === value;
    };
    const updateAmountOptions = () => {
      if (!amountFilter) return;
      const selected = amountFilter.value;
      const available = [...new Set(reportRows
        .filter((row) => reportFilters.filter((filter) => filter !== amountFilter).every((filter) => matchesFilter(row, filter)))
        .map((row) => Number(row.dataset.amount).toFixed(2)))].sort((a, b) => Number(a) - Number(b));
      amountFilter.replaceChildren(new Option('Hamısı', ''));
      available.forEach((amount) => amountFilter.add(new Option(amount, amount)));
      amountFilter.value = available.includes(selected) ? selected : '';
    };
    const applyReportFilters = () => {
      updateAmountOptions();
      let visibleCount = 0;
      let visibleTotal = 0;
      for (const row of reportRows) {
        const matches = reportFilters.every((filter) => matchesFilter(row, filter));
        row.hidden = !matches;
        if (matches) {
          visibleCount += 1;
          visibleTotal += Number(row.dataset.amount);
        }
      }
      if (totalOutput) totalOutput.textContent = visibleTotal.toFixed(2);
      if (emptyRow) emptyRow.hidden = visibleCount !== 0;
    };
    reportFilters.forEach((filter) => filter.addEventListener('change', applyReportFilters));
    applyReportFilters();
  }

  document.addEventListener('click', (event) => {
    document.querySelectorAll('[data-category-filter]').forEach((container) => {
      if (!container.contains(event.target)) {
        const filter = container.querySelector('[data-category-type-filter]');
        if (filter) filter.hidden = true;
      }
    });
  });
})();
