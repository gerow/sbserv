$(document).ready(function() {
      $("#dir-table").tablesorter({
        dateFormat: ""
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