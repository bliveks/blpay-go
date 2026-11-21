(() => {
  let lastOpener = null;
  let pageScrollLock = null;

  const lockPageScroll = () => {
    if (pageScrollLock) return;
    const body = document.body;
    const scrollY = window.scrollY;
    const scrollbarWidth = Math.max(0, window.innerWidth - document.documentElement.clientWidth);
    pageScrollLock = {
      scrollY,
      rootOverflow: document.documentElement.style.overflow,
      styles: {
        position: body.style.position,
        top: body.style.top,
        left: body.style.left,
        right: body.style.right,
        width: body.style.width,
        paddingRight: body.style.paddingRight,
      },
    };
    document.documentElement.style.overflow = 'hidden';
    body.style.position = 'fixed';
    body.style.top = `-${scrollY}px`;
    body.style.left = '0';
    body.style.right = '0';
    body.style.width = '100%';
    if (scrollbarWidth) body.style.paddingRight = `calc(${pageScrollLock.styles.paddingRight || '0px'} + ${scrollbarWidth}px)`;
  };

  const unlockPageScroll = () => {
    if (!pageScrollLock) return;
    const {scrollY, styles} = pageScrollLock;
    const body = document.body;
    document.documentElement.style.overflow = pageScrollLock.rootOverflow;
    Object.assign(body.style, styles);
    pageScrollLock = null;
    window.scrollTo(0, scrollY);
  };

  const pageLoading = document.getElementById('page-loading');
  const showPageLoading = () => {
    if (!pageLoading) return;
    pageLoading.classList.add('is-visible');
    pageLoading.setAttribute('aria-hidden', 'false');
  };
  const hidePageLoading = () => {
    pageLoading?.classList.remove('is-visible');
    pageLoading?.setAttribute('aria-hidden', 'true');
  };
  window.addEventListener('pageshow', hidePageLoading);
  document.addEventListener('click', (event) => {
    const link = event.target.closest('a[href]');
    if (!link || event.defaultPrevented || event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey || link.hasAttribute('download')) return;
    if (link.target && link.target !== '_self') return;
    const destination = new URL(link.href, window.location.href);
    if (destination.origin !== window.location.origin) return;
    if (destination.pathname === window.location.pathname && destination.search === window.location.search && destination.hash) return;
    if (destination.href !== window.location.href) window.setTimeout(showPageLoading, 0);
  });
  document.addEventListener('submit', (event) => {
    if (!event.defaultPrevented) window.setTimeout(showPageLoading, 0);
  });

  const validMonth = (value) => /^\d{4}-(0[1-9]|1[0-2])$/.test(value);
  const saveMonth = (value) => {
    if (!validMonth(value)) return;
    const secure = window.location.protocol === 'https:' ? '; Secure' : '';
    document.cookie = `blpay_month=${value}; Path=/; SameSite=Lax${secure}`;
  };
  const navigateWithoutMonthQuery = () => {
    const url = new URL(window.location.href);
    url.searchParams.delete('month');
    window.setTimeout(() => {
      showPageLoading();
      window.location.assign(`${url.pathname}${url.search}${url.hash}`);
    }, 0);
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
        lockPageScroll();
        dialog.showModal();
        const initialFocus = dialog.id === 'report-filter-modal'
          ? dialog.querySelector('[data-modal-close]')
          : dialog.querySelector('input:not([type="hidden"]), select, textarea');
        initialFocus?.focus({preventScroll: true});
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
    dialog.addEventListener('close', () => {
      if (!document.querySelector('dialog.modal[open]')) unlockPageScroll();
      lastOpener?.focus({preventScroll: true});
    });
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
    const amountFilters = reportFilters.filter((filter) => filter.dataset.reportFilter === 'amount');
    const totalOutput = document.querySelector('[data-report-total]');
    const matchesFilter = (row, filter) => {
      const value = filter.value;
      if (value === '__all__') return true;
      const field = filter.dataset.reportFilter;
      if (field === 'amount') return Number(row.dataset.amount).toFixed(2) === value;
      if (field === 'note' && value === '__empty__') return !row.dataset.note;
      return row.dataset[field] === value;
    };
    const updateAmountOptions = () => {
      if (!amountFilters.length) return;
      const selected = amountFilters[0].value;
      const available = [...new Set(reportRows
        .filter((row) => reportFilters.filter((filter) => filter.dataset.reportFilter !== 'amount').every((filter) => matchesFilter(row, filter)))
        .map((row) => Number(row.dataset.amount).toFixed(2)))].sort((a, b) => Number(a) - Number(b));
      amountFilters.forEach((filter) => {
        filter.replaceChildren(new Option('Hamısı', '__all__'));
        available.forEach((amount) => filter.add(new Option(amount, amount)));
        filter.value = available.includes(selected) ? selected : '__all__';
      });
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
          visibleTotal += Number(row.dataset.amount) * (row.dataset.kind === 'xerc' ? -1 : 1);
        }
      }
      if (totalOutput) totalOutput.textContent = visibleTotal.toFixed(2);
      if (emptyRow) emptyRow.hidden = visibleCount !== 0;
    };
    reportFilters.forEach((filter) => filter.addEventListener('change', () => {
      reportFilters.filter((other) => other.dataset.reportFilter === filter.dataset.reportFilter).forEach((other) => {
        other.value = filter.value;
      });
      applyReportFilters();
    }));
    document.querySelector('[data-report-filter-reset]')?.addEventListener('click', () => {
      reportFilters.forEach((filter) => {
        filter.value = '__all__';
      });
      applyReportFilters();
    });
    applyReportFilters();
  }

  document.addEventListener('click', (event) => {
    const toggle = event.target.closest('[data-report-note-toggle]');
    if (!toggle) return;
    const note = toggle.closest('.report-mobile-content')?.querySelector('.report-mobile-note');
    if (!note) return;
    note.hidden = !note.hidden;
    toggle.setAttribute('aria-expanded', String(!note.hidden));
    toggle.setAttribute('aria-label', note.hidden ? 'Qeydi göstər' : 'Qeydi gizlət');
  });

  document.addEventListener('click', (event) => {
    document.querySelectorAll('[data-category-filter]').forEach((container) => {
      if (!container.contains(event.target)) {
        const filter = container.querySelector('[data-category-type-filter]');
        if (filter) filter.hidden = true;
      }
    });
  });

  if (window.location.hash === '#quick-add') {
    const openTransactionButton = document.querySelector('[data-modal-open="transaction-modal"]');
    if (openTransactionButton) {
      window.history.replaceState({}, '', `${window.location.pathname}${window.location.search}`);
      openTransactionButton.click();
    }
  }
})();
