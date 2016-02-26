(function() {
  var newEl = document.createElement('script'),
      firstScriptTag = document.getElementsByTagName('script')[0];

  if (firstScriptTag) {
    newEl.async = 1;
    newEl.src = '//' + window.location.hostname + ':35729/livereload.js';
    firstScriptTag.parentNode.insertBefore(newEl, firstScriptTag);
  }

})();
