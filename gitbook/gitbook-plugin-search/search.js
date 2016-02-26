require([
    "gitbook",
    "lodash"
], function(gitbook, _) {
    var index = null;
    var $searchInput, $searchForm;

    // Use a specific index
    function loadIndex(data) {
        index = lunr.Index.load(data);
    }

    // Fetch the search index
    function fetchIndex() {
        $.getJSON(gitbook.state.basePath+"/search_index.json")
        .then(loadIndex);
    }

    // Search for a term and return results
    function search(q) {
        if (!index) return;

        var results = _.chain(index.search(q))
        .map(function(result) {
            var parts = result.ref.split("#")
            return {
                path: parts[0],
                hash: parts[1]
            }
        })
        .value();

        return results;
    }

    // Create search form
    function createForm(value) {
        if ($searchForm) $searchForm.remove();

        $searchForm = $('<div>', {
            'class': 'book-search',
            'role': 'search'
        });

        $searchInput = $('<input>', {
            'type': 'text',
            'class': 'form-control',
            'val': value,
            'placeholder': 'Type to search'
        });

        $searchInput.appendTo($searchForm);
        $searchForm.prependTo(gitbook.state.$book.find('.book-summary'));
    }

    // Return true if search is open
    function isSearchOpen() {
        return gitbook.state.$book.hasClass("with-search");
    }

    // Toggle the search
    function toggleSearch(_state) {
        if (isSearchOpen() === _state) return;

        gitbook.state.$book.toggleClass("with-search", _state);

        // If search bar is open: focus input
        if (isSearchOpen()) {
            gitbook.sidebar.toggle(true);
            $searchInput.focus();
        } else {
            $searchInput.blur();
            $searchInput.val("");
            gitbook.sidebar.filter(null);
        }
    }

    // Recover current search when page changed
    function recoverSearch() {
        var keyword = gitbook.storage.get("keyword", "");

        createForm(keyword);

        if (keyword.length > 0) {
            if(!isSearchOpen()) {
                toggleSearch();
            }
            gitbook.sidebar.filter(_.pluck(search(keyword), "path"));
        }
    };


    gitbook.events.bind("start", function(config) {
        // Pre-fetch search index and create the form
        fetchIndex();
        createForm();

        // Type in search bar
        $(document).on("keyup", ".book-search input", function(e) {
            var key = (e.keyCode ? e.keyCode : e.which);
            var q = $(this).val();

            if (key == 27) {
                e.preventDefault();
                toggleSearch(false);
                return;
            }
            if (q.length == 0) {
                gitbook.sidebar.filter(null);
                gitbook.storage.remove("keyword");
            } else {
                var results = search(q);
                gitbook.sidebar.filter(
                    _.pluck(results, "path")
                );
                gitbook.storage.set("keyword", q);
            }
        });

        // Create the toggle search button
        gitbook.toolbar.createButton({
            icon: 'fa fa-search',
            label: 'Search',
            position: 'left',
            onClick: toggleSearch
        });

        // Bind keyboard to toggle search
        gitbook.keyboard.bind(['f'], toggleSearch)
    });

    gitbook.events.bind("page.change", recoverSearch);
});


