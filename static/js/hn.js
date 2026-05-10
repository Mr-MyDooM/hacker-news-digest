function getEffectiveTheme() {
    var saved = document.documentElement.getAttribute('data-theme');
    if (saved) return saved;
    if (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches) return 'dark';
    return 'light';
}

function setThemeIcon(theme) {
    var icon = document.querySelector('#theme-toggle i');
    if (icon) icon.className = theme === 'dark' ? 'fa fa-sun-o' : 'fa fa-moon-o';
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
const comparators = {
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
};

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

function setupSortHandlers() {
    $('.sort-dropdown [data-sort]').click(function() {
        const sortBy = $(this).data('sort');
        if (!comparators[sortBy]) return false;
        const sortOrder = (last_sort_by === sortBy) ? 'asc' : 'desc';
        applyAndRenderSort(sortBy, sortOrder);
        last_sort_by = (sortOrder === 'desc') ? sortBy : '';
        $(this).closest('.dropdown').removeClass('open');
        return false;
    });
}

function setupFilterHandlers() {
    $('.filter-dropdown [data-filter]').click(function() {
        const raw = $(this).data('filter');
        const topN = parseInt(raw);
        applyAndRenderFilter(topN);
        updateUrlHash({filter: isNaN(topN) ? '' : topN});
        $(this).closest('.dropdown').removeClass('open');
        return false;
    });
}

function PreviewImage(src) {
    $('#img-preview-modal img').attr('src', src);
    $('#img-preview-modal').modal();
}

$(function() {
    setupSortHandlers();
    setupFilterHandlers();

    // Restore state from URL hash
    const urlParams = new URLSearchParams(window.location.hash.substring(1));

    const filterBy = urlParams.get('filter');
    if (filterBy) applyAndRenderFilter(parseInt(filterBy));

    const sortBy = urlParams.get('sort');
    const sortOrder = urlParams.get('order') || 'desc';
    if (sortBy && comparators[sortBy]) {
        applyAndRenderSort(sortBy, sortOrder);
        last_sort_by = (sortOrder === 'desc') ? sortBy : '';
    }
});

$.scrollUp({
    scrollTrigger: '<i class="fa fa-chevron-circle-up fa-3x" id="scrollUp"></i>',
    scrollTitle: 'Scroll to top'
});

// Feature image modal
$('.post-item .post-summary .feature-image').click(function(e) {
    PreviewImage($('img', this).attr('src'));
    return false;
});

// Share: Web Share API with clipboard fallback
$('.post-item .share-icon').click(function(e) {
    const slug = $(this).data('slug');
    if (!slug) return false;
    const url = window.location.origin + window.location.pathname + '#' + slug;
    const title = $(this).closest('.post-item').find('.post-title a').text().trim();
    if (navigator.share) {
        navigator.share({title: title, url: url}).catch(function() {});
    } else if (navigator.clipboard) {
        navigator.clipboard.writeText(url).then(function() {
            const icon = $('.share-icon[data-slug="' + slug + '"] i');
            icon.removeClass('fa-share-alt').addClass('fa-check');
            setTimeout(() => icon.removeClass('fa-check').addClass('fa-share-alt'), 1500);
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
        if (secs < 60)  return secs + 's ago';
        var mins = Math.floor(secs / 60);
        if (mins < 60)  return mins + 'm ago';
        var hrs = Math.floor(mins / 60);
        if (hrs < 24)   return hrs + 'h ago';
        var days = Math.floor(hrs / 24);
        if (days < 30)  return days + 'd ago';
        return null; // fall back to absolute
    }
    document.querySelectorAll('.summit-time[data-submitted]').forEach(function(el) {
        var rel = timeAgo(el.getAttribute('data-submitted'));
        if (rel) {
            el.title = el.textContent.trim();
            el.textContent = rel;
        }
    });
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
        searchBox.classList.toggle('hidden');
        if (!searchBox.classList.contains('hidden')) {
            searchInput.focus();
        } else {
            searchInput.value = '';
            doSearch('');
        }
    });

    searchClose.addEventListener('click', function() {
        searchBox.classList.add('hidden');
        searchInput.value = '';
        doSearch('');
    });

    var searchTimer;
    searchInput.addEventListener('input', function() {
        clearTimeout(searchTimer);
        searchTimer = setTimeout(function() {
            doSearch(searchInput.value.trim());
        }, 150);
    });

    function doSearch(query) {
        var articles = document.querySelectorAll('.post-item:not(.ad)');
        if (!query) {
            articles.forEach(function(a) {
                a.classList.remove('search-hidden', 'search-match');
            });
            return;
        }
        var lower = query.toLowerCase();
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
        });
    }
})();
