$(".channel-paste").click(function() {
  $("#channelID").attr("value", $(this).attr("info"));
})

$(window).scroll(function() {
  if ($(document).scrollTop() > 0) {
    $('.scrollup').fadeIn('fast');
  } else {
    $('.scrollup').fadeOut('fast');
  }
})

$('.scrollup').click(function() {
  window.scroll(0, 0);
})

new Clipboard('.jscopy');
