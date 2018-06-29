$(document).ready(function() {
      $("#dir").tablesorter({
        showProcessing: true,
        ignoreCase: true,
        sortInitialOrder: "desc",
        sortReset: true,
        widgets: ['saveSort', 'sort2Hash'],
        widgetOptions: {
            saveSort: true
        },
    });
});

$(".playMovieButton").on('click', function(e){
    e.preventDefault();
    $(this).closest(".fileEntry").find(".moviePanel").collapse("toggle");
});

$('.moviePanel').on('hidden.bs.collapse', function () {
  $(this).empty();
})

$('.moviePanel').on('show.bs.collapse', function () {
  var src = $(this).attr('src');
  var type = $(this).attr('type');
  $(this).html("<video controls><source src='" + src + "' type='"+ type +"'></video>");
})

$('.dlfiles-div').on('click', function (){
  var downloadNodes = document.querySelectorAll('.downloadFileLink');
  for (var i = 0; i < downloadNodes.length; i++) {
    downloadNodes[i].click();
  }
});

function DownloadAllFiles() {
  var downloadNodes = document.querySelectorAll('.downloadFileLink');
  for (var i = 0; i < downloadNodes.length; i++) {
    downloadNodes[i].click();
  }
  return downloadNodes;
}
