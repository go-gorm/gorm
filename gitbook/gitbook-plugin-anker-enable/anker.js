require(["gitbook"], function(gitbook) {
    gitbook.events.bind("page.change", function() {
        // For Anker tag link
        if (location.hash) {
            // deley for other event
            setTimeout(function(){
                // wait for loading
                document.location = location.hash;
            }, 300);
        }
    });
});
