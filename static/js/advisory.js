// Shared temporal weather advisory controls
(function () {
  const STORAGE_KEY = 'weatherAdvisoryDismissed';
  const MOTION_QUERY = window.matchMedia
    ? window.matchMedia('(prefers-reduced-motion: reduce)')
    : { matches: false, addEventListener: null, addListener: null };
  let lightningTimer = null;

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

  function scheduleLightning() {
    if (MOTION_QUERY.matches || document.hidden) {
      return;
    }
    const overlay = document.getElementById('temporalLightning');
    if (!overlay) {
      return;
    }
    if (lightningTimer) {
      clearTimeout(lightningTimer);
    }
    const delay = 7000 + Math.random() * 8000;
    lightningTimer = window.setTimeout(triggerLightning, delay);
  }

  function triggerLightning() {
    const overlay = document.getElementById('temporalLightning');
    if (!overlay) {
      return;
    }
    overlay.classList.remove('strike');
    void overlay.offsetWidth; // force reflow so animation restarts
    overlay.classList.add('strike');
    window.setTimeout(function () {
      overlay.classList.remove('strike');
      scheduleLightning();
    }, 700);
  }

  function cancelLightning() {
    if (lightningTimer) {
      clearTimeout(lightningTimer);
      lightningTimer = null;
    }
  }

  function handleMotionPreferenceChange(event) {
    if (event.matches) {
      cancelLightning();
    } else {
      scheduleLightning();
    }
  }

  if (MOTION_QUERY.addEventListener) {
    MOTION_QUERY.addEventListener('change', handleMotionPreferenceChange);
  } else if (MOTION_QUERY.addListener) {
    MOTION_QUERY.addListener(handleMotionPreferenceChange);
  }

  document.addEventListener('visibilitychange', function () {
    if (document.hidden) {
      cancelLightning();
    } else {
      scheduleLightning();
    }
  });

  document.addEventListener('DOMContentLoaded', function () {
    window.checkWeatherAdvisoryDismissal();
    scheduleLightning();
  });
})();

