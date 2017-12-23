(function() {
  var links = document.links;
  for (var i = 0; i < links.length; i++) {
    var link = links[i];
    var m = link.href.match(/^http:\/\/friendfeed\.com\/([a-z0-9-]+)/);
    if (m !== null) {
      if (userNames.includes(m[1])) {
        link.href = link.href.replace(m[0], '/' + m[1]);
      } else {
        link.classList.add('no-user');
        link.href = link.href.replace(m[0], 'https://freefeed.net/' + m[1]);
        link.target = '_blank';
        if (m[1] === 'search') {
          link.href = link.href.replace('?q=', '?qs=');
        }
      }
    }
  }

  var topDiv = document.createElement('div');
  topDiv.className = 'index-link-block';
  topDiv.appendChild(document.createTextNode('('))
  var a = topDiv.appendChild(document.createElement('a'))
  topDiv.appendChild(document.createTextNode(')'))
  a.href = '/';
  a.textContent = 'all archives';
  document.body.insertBefore(topDiv, document.body.firstChild);
})();