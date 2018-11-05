function drawRandPoem() {
    $.getJSON('/poem/random', function (data) {
        var poemArea = document.getElementById('rnd_poem');
        var paras = data.Paragraphs;
        var author = data.Author;
        var title = data.Title;
        var shown = Math.min(1, paras.length);
        var index = getRandom(0, paras.length - shown + 1);
        var body = "";
        for (i = 0; i < shown; i++) {
            body += paras[index + i];
        };
        poemArea.innerHTML = "<span style='font-size:78%'>" + body + "</span>" +
            "<span style='font-size:60%'>(" + author + ")</span>";
    });
};

function getRandom(start, end) {
    var range = end - start;
    var num = parseInt(Math.random() * range + start);
    return num;
};