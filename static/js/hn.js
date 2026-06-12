function getEffectiveTheme() {
    var saved = document.documentElement.getAttribute('data-theme');
    if (saved) return saved;
    if (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches) return 'dark';
    return 'light';
}

function setThemeIcon(theme) {
    var icon = document.querySelector('#theme-toggle i');
    var btn = document.querySelector('#theme-toggle');
    if (icon) icon.className = theme === 'dark' ? 'fa fa-sun-o' : 'fa fa-moon-o';
    if (btn) btn.setAttribute('aria-label', theme === 'dark' ? 'Switch to light theme' : 'Switch to dark theme');
}

// Sync icon on load
setThemeIcon(getEffectiveTheme());

// React to system theme changes (when user hasn't manually toggled)
if (window.matchMedia) {
    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', function(e) {
        if (!localStorage.getItem('theme')) {
            setThemeIcon(e.matches ? 'dark' : 'light');
        }
    });
}

$('#theme-toggle').click(function(e) {
    e.preventDefault();
    var current = getEffectiveTheme();
    var next = current === 'dark' ? 'light' : 'dark';
    document.documentElement.setAttribute('data-theme', next);
    localStorage.setItem('theme', next);
    setThemeIcon(next);
});

let last_sort_by = 'rank';

function updateUrlHash(newParams) {
    const params = new URLSearchParams(window.location.hash.substring(1));
    for (const [key, value] of Object.entries(newParams)) {
        if (value !== null && value !== undefined && value !== '') {
            params.set(key, value);
        } else {
            params.delete(key);
        }
    }
    if (history.replaceState) {
        history.replaceState(null, null, '#' + params.toString());
    } else {
        window.location.hash = params.toString();
    }
}

// Read sort values from data-* attrs on article element (avoids brittle DOM text parsing)
var comparators = Object.assign(Object.create(null), {
    'rank': (a, b) => parseInt($(a).data('rank')) - parseInt($(b).data('rank')),
    'score': (a, b) => {
        const diff = parseInt($(b).data('points')) - parseInt($(a).data('points'));
        return diff !== 0 ? diff : parseInt($(a).data('rank')) - parseInt($(b).data('rank'));
    },
    'comments': (a, b) => {
        const diff = parseInt($(b).data('comments')) - parseInt($(a).data('comments'));
        return diff !== 0 ? diff : parseInt($(a).data('rank')) - parseInt($(b).data('rank'));
    },
    'time': (a, b) => {
        const diff = parseInt($(b).data('time')) - parseInt($(a).data('time'));
        return diff !== 0 ? diff : parseInt($(a).data('rank')) - parseInt($(b).data('rank'));
    }
});

function applyAndRenderSort(sortBy, sortOrder) {
    const comparator = comparators[sortBy];
    if (!comparator) return;

    const articles = $('article');
    const items = articles.get();

    const ads = [];
    items.forEach(function(item, index) {
        if ($(item).hasClass('ad')) ads.push({index: index, element: item});
    });

    const newsItems = items.filter(item => !$(item).hasClass('ad'));
    newsItems.sort(function(a, b) {
        const result = comparator(a, b);
        return sortOrder === 'asc' ? -result : result;
    });

    ads.forEach(function(ad) {
        newsItems.splice(ad.index, 0, ad.element);
    });

    articles.detach();
    $(newsItems).insertBefore($('footer'));

    updateUrlHash({sort: sortBy, order: sortOrder});
}

function applyAndRenderFilter(topN) {
    if (!topN || topN <= 0 || isNaN(topN)) {
        $('article').show();
        return;
    }

    const scores = $.map($('article:not(.ad)'), e => parseInt($(e).data('points')) || 0)
        .sort((a, b) => b - a);

    const threshold = topN < scores.length ? scores[topN - 1] : 0;

    $('article').each(function() {
        if ($(this).hasClass('ad')) return;
        $(this).toggle((parseInt($(this).data('points')) || 0) >= threshold);
    });
}

function updateDropdownActive($container, value, dataAttr) {
    $container.find('li').removeClass('active').find('a').removeAttr('aria-current');
    if (value) {
        const $activeLi = $container.find(`[data-${dataAttr}="${value}"]`).parent('li');
        $activeLi.addClass('active').find('a').attr('aria-current', 'true');
    }
}

function setupSortHandlers() {
    $('.sort-dropdown [data-sort]').click(function() {
        const sortBy = $(this).data('sort');
        if (!comparators[sortBy]) return false;
        const sortOrder = (last_sort_by === sortBy) ? 'asc' : 'desc';
        applyAndRenderSort(sortBy, sortOrder);
        last_sort_by = (sortOrder === 'desc') ? sortBy : '';
        updateDropdownActive($('.sort-dropdown'), sortBy, 'sort');
        $(this).closest('.dropdown').removeClass('open');
        return false;
    });
}

function setupArchiveHandlers() {
    $('#more-archive').click(function(e) {
        e.preventDefault();
        e.stopPropagation();
        $('.archive-item').removeClass('hidden');
        $('#archive-divider').remove();
        $(this).parent().remove();
    });
}

function setupFilterHandlers() {
    $('.filter-dropdown [data-filter]').click(function() {
        const raw = $(this).data('filter');
        const topN = parseInt(raw);
        applyAndRenderFilter(topN);
        updateUrlHash({filter: isNaN(topN) ? '' : topN});
        updateDropdownActive($('.filter-dropdown'), raw, 'filter');
        $(this).closest('.dropdown').removeClass('open');
        return false;
    });
}

function PreviewImage(src, title) {
    var $img = $('#modal-image');
    $img.attr('src', src);
    if (title) {
        $img.attr('alt', 'Full size feature image for ' + title);
    } else {
        $img.attr('alt', 'Full size feature image');
    }
    $('#img-preview-modal').modal();
}

$(function() {
    setupSortHandlers();
    setupFilterHandlers();
    setupArchiveHandlers();

    // Set defaults
    updateDropdownActive($('.sort-dropdown'), 'rank', 'sort');
    updateDropdownActive($('.filter-dropdown'), 'all', 'filter');

    // Restore state from URL hash
    const urlParams = new URLSearchParams(window.location.hash.substring(1));

    const filterBy = urlParams.get('filter');
    if (filterBy) {
        applyAndRenderFilter(parseInt(filterBy));
        updateDropdownActive($('.filter-dropdown'), filterBy, 'filter');
    }

    const sortBy = urlParams.get('sort');
    const sortOrder = urlParams.get('order') || 'desc';
    if (sortBy && comparators[sortBy]) {
        applyAndRenderSort(sortBy, sortOrder);
        last_sort_by = (sortOrder === 'desc') ? sortBy : '';
        updateDropdownActive($('.sort-dropdown'), sortBy, 'sort');
    }
});

$.scrollUp({
    scrollTrigger: '<i class="fa fa-chevron-circle-up fa-3x" id="scrollUp"></i>',
    scrollTitle: 'Scroll to top'
});

// Navbar auto-hide on scroll
(function() {
    var lastY = window.scrollY;
    var navbar = document.querySelector('.navbar.navbar-fixed-top');
    if (!navbar) return;
    window.addEventListener('scroll', function() {
        var currentY = window.scrollY;
        if (currentY > lastY && currentY > 80) {
            navbar.classList.add('nav-hidden');
        } else {
            navbar.classList.remove('nav-hidden');
        }
        lastY = currentY;
    }, { passive: true });
})();

// Feature image modal
$('.post-item .post-summary').on('click', '.feature-image', function(e) {
    var $img = $(this).find('img');
    var src = $img.attr('src');
    var title = $(this).closest('.post-item').find('.post-title a').text().trim();
    if (src) PreviewImage(src, title);
    return false;
});

// Share: Web Share API with clipboard fallback
$('.post-item .share-icon').click(function(e) {
    const slug = $(this).data('slug');
    if (!slug) return false;
    const url = window.location.origin + window.location.pathname + '#' + slug;
    const title = $(this).closest('.post-item').find('.post-title a').text().trim();
    const btn = $(this);
    if (navigator.share) {
        navigator.share({title: title, url: url}).catch(function() {});
    } else if (navigator.clipboard) {
        navigator.clipboard.writeText(url).then(function() {
            const icon = btn.find('i');
            const originalLabel = btn.attr('aria-label');
            icon.removeClass('fa-share-alt').addClass('fa-check');
            btn.attr('aria-label', 'Permalink copied!');
            setTimeout(() => {
                icon.removeClass('fa-check').addClass('fa-share-alt');
                btn.attr('aria-label', originalLabel);
            }, 1500);
        });
    }
    return false;
});

// Switch images to eager after 30s (reduces initial load)
setTimeout(() => $('.post-item img').attr('loading', 'eager'), 30000);

// Relative time for submit times
(function() {
    function timeAgo(dateStr) {
        var d = new Date(dateStr);
        if (isNaN(d)) return null;
        var secs = Math.floor((Date.now() - d) / 1000);
        if (secs < 60)  return plural(secs, 'second') + ' ago';
        var mins = Math.floor(secs / 60);
        if (mins < 60)  return plural(mins, 'minute') + ' ago';
        var hrs = Math.floor(mins / 60);
        if (hrs < 24)   return plural(hrs, 'hour') + ' ago';
        var days = Math.floor(hrs / 24);
        if (days < 7)   return plural(days, 'day') + ' ago';
        var weeks = Math.floor(days / 7);
        if (weeks < 5)  return plural(weeks, 'week') + ' ago';
        var months = Math.floor(days / 30);
        if (months < 12) return plural(months, 'month') + ' ago';
        var years = Math.floor(days / 365);
        return plural(years, 'year') + ' ago';
    }
    function plural(n, word) {
        return n + ' ' + (n === 1 ? word : word + 's');
    }
    document.querySelectorAll('.summit-time[data-submitted]').forEach(function(el) {
        var rel = timeAgo(el.getAttribute('data-submitted'));
        if (rel) {
            var textNode = el.querySelector('.time-ago-text');
            if (textNode) {
                textNode.textContent = rel;
            } else {
                el.textContent = rel;
            }
        }
    });
    var lu = document.querySelector('.last-updated[data-updated]');
    if (lu) {
        function refreshLastUpdated() {
            var rel = timeAgo(lu.getAttribute('data-updated'));
            if (rel) lu.textContent = rel;
        }
        refreshLastUpdated();
        setInterval(refreshLastUpdated, 60000);
    }
})();

// =============================================
// READING TIME
// =============================================
(function() {
    document.querySelectorAll('.reading-time').forEach(function(el) {
        var slug = el.getAttribute('data-slug');
        var body = document.querySelector('.summary-body[data-slug="' + slug + '"]');
        if (!body) return;
        var text = body.querySelector('.summary-text');
        if (!text) return;
        var words = text.textContent.trim().split(/\s+/).length;
        if (words < 10) return;
        var min = Math.max(1, Math.round(words / 200));
        el.textContent = min + ' min read';
    });
})();

// =============================================
// COLLAPSIBLE SUMMARIES
// =============================================
(function() {
    var checkOverflow = function() {
        document.querySelectorAll('.summary-body').forEach(function(body) {
            var text = body.querySelector('.summary-text');
            var toggle = body.querySelector('.summary-toggle');
            if (!text || !toggle) return;
            if (text.scrollHeight > text.clientHeight) {
                toggle.style.display = 'inline-block';
            }
        });
    };
    // Check after images load (can affect layout)
    checkOverflow();
    setTimeout(checkOverflow, 500);
    window.addEventListener('load', checkOverflow);

    document.querySelectorAll('.summary-toggle').forEach(function(btn) {
        btn.addEventListener('click', function(e) {
            var body = this.closest('.summary-body');
            if (!body) return;
            var expanded = body.classList.toggle('expanded');
            this.textContent = expanded ? 'Show less' : 'Show more';
            this.setAttribute('aria-expanded', expanded);
        });
    });
})();

// =============================================
// KEYBOARD SHORTCUTS
// =============================================
(function() {
    var items = [];
    var current = -1;

    function refreshItems() {
        items = document.querySelectorAll('.post-item:not(.ad):not(.search-hidden)');
    }

    function scrollToItem(idx) {
        if (idx < 0 || idx >= items.length) return;
        current = idx;
        items.forEach(function(el, i) {
            el.classList.toggle('active-keyboard', i === idx);
        });
        items[idx].scrollIntoView({behavior: 'smooth', block: 'center'});
        // Focus the link for screen readers to announce the selection
        const link = items[idx].querySelector('.post-title a');
        if (link) link.focus({preventScroll: true});
    }

    document.addEventListener('keydown', function(e) {
        if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA') {
            if (e.key === 'Escape') {
                document.getElementById('search-input').blur();
                if (document.getElementById('search-box')) {
                    document.getElementById('search-box').classList.add('hidden');
                }
            }
            return;
        }

        refreshItems();
        if (items.length === 0) return;

        switch (e.key) {
            case 'j':
                e.preventDefault();
                scrollToItem(current < items.length - 1 ? current + 1 : 0);
                break;
            case 'k':
                e.preventDefault();
                scrollToItem(current > 0 ? current - 1 : items.length - 1);
                break;
            case 'o':
                e.preventDefault();
                if (current >= 0 && current < items.length) {
                    var link = items[current].querySelector('.post-title a');
                    if (link) window.open(link.href, '_blank');
                }
                break;
            case 'c':
                e.preventDefault();
                if (current >= 0 && current < items.length) {
                    var commentLink = items[current].querySelector('.comment a');
                    if (commentLink) window.open(commentLink.href, '_blank');
                }
                break;
            case '/':
                e.preventDefault();
                var searchToggle = document.getElementById('search-toggle');
                if (searchToggle) searchToggle.click();
                break;
        }
    });
})();

// =============================================
// SEARCH
// =============================================
(function() {
    var searchToggle = document.getElementById('search-toggle');
    var searchBox = document.getElementById('search-box');
    var searchInput = document.getElementById('search-input');
    var searchClose = document.getElementById('search-close');
    if (!searchToggle || !searchBox || !searchInput) return;

    searchToggle.addEventListener('click', function(e) {
        e.preventDefault();
        const hidden = searchBox.classList.toggle('hidden');
        this.setAttribute('aria-expanded', !hidden);
        if (!hidden) {
            searchInput.focus();
        } else {
            searchInput.value = '';
            doSearch('');
        }
    });

    searchClose.addEventListener('click', function() {
        searchBox.classList.add('hidden');
        searchToggle.setAttribute('aria-expanded', 'false');
        searchInput.value = '';
        doSearch('');
        searchToggle.focus();
    });

    var searchClearBtn = document.getElementById('search-clear-btn');
    if (searchClearBtn) {
        searchClearBtn.addEventListener('click', function() {
            searchInput.value = '';
            doSearch('');
            searchInput.focus();
        });
    }

    var searchTimer;
    searchInput.addEventListener('input', function() {
        clearTimeout(searchTimer);
        searchTimer = setTimeout(function() {
            doSearch(searchInput.value.trim());
        }, 150);
    });

    function doSearch(query) {
        var articles = document.querySelectorAll('.post-item:not(.ad)');
        var noResults = document.getElementById('search-no-results');
        if (!query) {
            articles.forEach(function(a) {
                a.classList.remove('search-hidden', 'search-match');
            });
            if (noResults) noResults.classList.add('hidden');
            return;
        }
        var lower = query.toLowerCase();
        var matchCount = 0;
        articles.forEach(function(a) {
            var title = a.querySelector('.post-title a');
            var summary = a.querySelector('.summary-text');
            var titleText = title ? title.textContent.toLowerCase() : '';
            var summaryText = summary ? summary.textContent.toLowerCase() : '';
            var author = a.querySelector('.author-link a');
            var authorText = author ? author.textContent.toLowerCase() : '';
            var domain = a.querySelector('.host');
            var domainText = domain ? domain.textContent.toLowerCase() : '';
            var match = titleText.indexOf(lower) !== -1 || summaryText.indexOf(lower) !== -1 ||
                        authorText.indexOf(lower) !== -1 || domainText.indexOf(lower) !== -1;
            a.classList.toggle('search-hidden', !match);
            a.classList.toggle('search-match', match);
            if (match) matchCount++;
        });
        if (noResults) {
            noResults.classList.toggle('hidden', matchCount > 0);
        }
    }
})();
