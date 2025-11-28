// Shared temporal weather advisory controls
(function () {
  const STORAGE_KEY = 'weatherAdvisoryDismissed';

  function getElements() {
    return {
      advisory: document.getElementById('chaosWeatherAdvisory'),
      indicator: document.getElementById('weatherIndicator'),
    };
  }

  function hideAdvisory(advisory, indicator) {
    if (advisory) {
      advisory.style.display = 'none';
    }
    if (indicator) {
      indicator.style.display = 'block';
    }
  }

  function showAdvisory(advisory, indicator) {
    if (advisory) {
      advisory.style.display = 'block';
    }
    if (indicator) {
      indicator.style.display = 'none';
    }
  }

  window.dismissWeatherAdvisory = function () {
    const { advisory, indicator } = getElements();
    hideAdvisory(advisory, indicator);
    try {
      localStorage.setItem(STORAGE_KEY, 'true');
    } catch (_) {
      /* ignore storage errors */
    }
  };

  window.restoreWeatherAdvisory = function () {
    const { advisory, indicator } = getElements();
    showAdvisory(advisory, indicator);
    try {
      localStorage.removeItem(STORAGE_KEY);
    } catch (_) {
      /* ignore storage errors */
    }
  };

  window.checkWeatherAdvisoryDismissal = function () {
    const { advisory, indicator } = getElements();
    if (!advisory) {
      return;
    }
    let dismissed = false;
    try {
      dismissed = localStorage.getItem(STORAGE_KEY) === 'true';
    } catch (_) {
      dismissed = false;
    }
    if (dismissed) {
      hideAdvisory(advisory, indicator);
    } else {
      showAdvisory(advisory, indicator);
    }
  };

  document.addEventListener('DOMContentLoaded', function () {
    window.checkWeatherAdvisoryDismissal();
  });
})();

