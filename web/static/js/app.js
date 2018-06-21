var targetFile;
var reader = new FileReader();
var targetFileData = ""; // Store target file's content, if any.

$( document ).ready(function() {

  $("form#download-form").submit(function( event ) {
    event.preventDefault();
    var downloadOptions = gatherOptions()
    apiDownload(downloadOptions);
  });

  $("form#login-form").submit(function( event ) {
    event.preventDefault();
    var username = $("#login-username-input").val().trim()
    var password = $("#login-password-input").val().trim()
    $("#login-modal").removeClass('is-active').addClass('is-clipped')
    apiLogin(username, password)
  });

  $("#mode-select").on( "change", function() {
    var value = this.value.toLowerCase();

    if (value === "links") {
      $("#target-is-self-checkbox-container").hide();
      $("#target-file-input-container").fadeIn("slow");
    } else {
      $("#target-file-input-container").hide();
      $("#target-is-self-checkbox-container").fadeIn("slow");
    }
  });

  // Grab the file and set its content to our variable
  function prepareUpload(event)
  {
    targetFile = event.target.files[0];

    // Closure to capture the file content.
    reader.onload = (function(theFile) {
      return function(e) {targetFileData = e.target.result};
    })(targetFile);

    reader.readAsText(targetFile);
  }

  // init page elements before doing anything
  initPageElements();


  $('input[type=file]').on('change', prepareUpload); // event listener for the target file

  $("#insert-creds-button").on( "click", function() {
    $("#login-modal").removeClass('is-clipped').addClass('is-active')
  });


  $("#close-notification-btn").on( "click", function() {
    $("#notifications").fadeOut("slow")
  });

  $("#login-cancel,#modal-close-button").on( "click", function() {
    $("#login-modal").removeClass('is-active').addClass('is-clipped')
  });

  $("#login-logout-button").on( "click", function() {
    $("#login-username-input").val("")
    $("#login-password-input").val("")
    $("#login-modal").removeClass('is-active').addClass('is-clipped')
    apiLogout()
  });

  $("#stop-download-button").on('click', function() {
    apiStopDownload()
  })


});


var gatherOptions = function() {

  var mode = $("#mode-select").val().toLowerCase();
  return {
    'crawlID': $("#crawlID-input").val().trim(),
    'mode': mode,
    'filter': $("#filter-input").val().trim(),
    'order': $("#order-input").val().trim(),
    'resume': !$("#do-not-resume-checkbox").is(':checked'),
    'details': !$("#hide-details-checkbox").is(':checked'),
    'target': targetFileData,
    "output": $("#output-filepath-input").val().trim(),
    // credential override:
    'username': $("#custom-username-input").val().trim(),
    'password': $("#custom-password-input").val().trim()
  }
}


var initPageElements = function() {
  $("#mode-select").trigger("change");
}

var webSocketURL = "ws://" + window.location.host + "/progress";

function start(webSocketURL){
  ws = new WebSocket(webSocketURL);
  ws.onmessage = function(evt) {
    message = JSON.parse(evt.data)
    $("progress").attr('value', message.progressPercentage)
    $("#ETA").html("ETA: " + message.ETA)
    $("#totalElements").html("Total Elements: " + message.totalElements)
    $("#chunkSize").html("Chunk Size: " + message.chunkSize)
    $("#doneElements").html("Done Elements: " + message.doneElements)
    $("#errors").html("Errors: " + message.errorsCount)

  };
  ws.onclose = function(){
    ws = undefined
    // Try to reconnect in 2 seconds
    setTimeout(function(){start(webSocketURL)}, 2000);
  };
}

start(webSocketURL)
