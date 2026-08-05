/*
 * Failure states for every region this site fills from the statistics service.
 *
 * Three regions use it today -- the traffic table and the dashboard tiles on
 * the home page, and the repository detail on each /site/<key>/ page -- and
 * they get their loading, timeout, unreachable and HTTP-error behaviour from
 * one implementation, so the three can never drift apart.
 *
 * The service sleeps when nobody is looking at this page (fly.io,
 * auto_stop_machines), so "slow" and "gone" are both ordinary states here, not
 * exceptional ones. Each is named in words the visitor can act on, and each
 * says plainly that the arc42 sites themselves are fine -- only this page's
 * numbers are missing.
 *
 * The page sets window.ARC42_STATS_API before loading this file; it is the only
 * thing that varies between production (fly.io) and `make site` (localhost).
 */
(function () {
    'use strict';

    var API = window.ARC42_STATS_API;

    if (!window.htmx || !API) { return; }

    // htmx 1.x has no hx-timeout attribute; without this a hanging service
    // would leave the loading state on screen forever.
    htmx.config.timeout = 15000;

    // name the host the page is actually talking to -- fly.io in production,
    // localhost when served by `make site` against a local backend
    var HOST = API.replace(/^https?:\/\//, '').replace(/\/.*$/, '');

    var SITES_FINE = 'The arc42 sites themselves are unaffected — only this is missing.';
    var SLEEPS = 'The statistics service sleeps when nobody is looking at this page, so the first request of the day can need a moment.';

    function icon() {
        return '<svg class="stats-panel__icon" viewBox="0 0 24 24" fill="none" ' +
               'stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true">' +
               '<path d="M12 3 1.8 20.5h20.4L12 3Z"/><path d="M12 9.5v5"/><path d="M12 18h.01"/></svg>';
    }

    function wire(regionId, slotId, path, noun) {
        var region = document.getElementById(regionId);
        if (!region) { return; }

        var endpoint = API + path;

        function showPanel(title, text, detail) {
            var slot = document.getElementById(slotId);
            if (!slot) { return; }
            slot.innerHTML =
                '<div class="stats-panel" role="alert">' + icon() +
                '<div class="stats-panel__body">' +
                '<p class="stats-panel__title">' + title + '</p>' +
                '<p class="stats-panel__text">' + text + '</p>' +
                '<p class="stats-panel__detail">' + detail + '</p>' +
                '<button type="button" class="stats-panel__retry">Try again</button>' +
                '</div></div>';
            region.setAttribute('aria-busy', 'false');
        }

        region.addEventListener('htmx:timeout', function () {
            showPanel('The statistics service did not answer',
                      HOST + ' took longer than 15 seconds to send the ' + noun + '. ' + SITES_FINE,
                      SLEEPS);
        });

        region.addEventListener('htmx:sendError', function () {
            showPanel('The statistics service is unreachable',
                      'Your browser could not reach ' + HOST + ' at all. ' + SITES_FINE,
                      'This is either a network problem on your side, or the service is down.');
        });

        region.addEventListener('htmx:responseError', function (event) {
            var status = (event.detail && event.detail.xhr) ? event.detail.xhr.status : 0;
            showPanel('The statistics service reported a problem',
                      HOST + ' answered with HTTP ' + status + '. ' + SITES_FINE,
                      'Nothing you can do from here; the service needs a look.');
        });

        region.addEventListener('htmx:afterSwap', function () {
            region.setAttribute('aria-busy', 'false');
        });

        region.addEventListener('click', function (event) {
            var button = event.target.closest('.stats-panel__retry');
            if (!button) { return; }
            button.disabled = true;
            button.textContent = 'Trying…';
            region.setAttribute('aria-busy', 'true');
            htmx.ajax('GET', endpoint, { target: '#' + slotId, swap: 'outerHTML' });
        });
    }

    window.arc42Status = { wire: wire };
}());
