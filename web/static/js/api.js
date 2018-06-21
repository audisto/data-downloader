var apiDownload = function(downloadOptions){
  $.ajax({
    url: '/download',
    type: 'POST',
    data: JSON.stringify(downloadOptions),
    cache: false,
    dataType: 'json',
    processData: false, // Don't process the files
    contentType: false, // Set content type to false as jQuery will tell the server its a query string request
    success: function(data, textStatus, jqXHR)
    {
      $("#notifications").removeClass('is-danger').addClass('is-success');
      $("#notifications-content").html(data.message)
      $("#notifications").fadeIn("slow")
    },
    error: function(jqXHR, textStatus, errorThrown)
    {
      $("#notifications").removeClass('is-success').addClass('is-danger');
      $("#notifications-content").html(jqXHR.responseJSON.error)
      $("#notifications").fadeIn("slow")
    }
  });
}

var apiStopDownload = function() {
  $.ajax({
    url: '/stop',
    type: 'POST',
    dataType: 'json',
    processData: false, // Don't process the files
    contentType: false, // Set content type to false as jQuery will tell the server its a query string request
    success: function(data, textStatus, jqXHR)
    {
      $("#notifications").removeClass('is-danger').addClass('is-success')
      $("#notifications-content").html(data.message)
      $("#notifications").fadeIn("slow")
    },
    error: function(jqXHR, textStatus, errorThrown)
    {
      ("#notifications").removeClass('is-success').addClass('is-danger')
      $("#notifications-content").html(jqXHR.responseJSON.error)
      $("#notifications").fadeIn("slow")
    }
  });
}

var apiLogin = function(username, password) {
  $.ajax({
    url: '/login',
    type: 'POST',
    data: JSON.stringify({'username': username, 'password': password}),
    dataType: 'json',
    contentType: false,
    success: function(data, textStatus, jqXHR)
    {
      $("#notifications").removeClass('is-danger').addClass('is-success');
      $("#notifications-content").html(data.message)
      $("#notifications").fadeIn("slow")
    },
    error: function(jqXHR, textStatus, errorThrown)
    {
      $("#notifications").removeClass('is-success').addClass('is-danger');
      $("#notifications-content").html(jqXHR.responseJSON.error)
      $("#notifications").fadeIn("slow")
    }
  });
}


var apiLogout = function() {
  $.ajax({
    url: '/logout',
    type: 'GET',
    dataType: 'json',
    contentType: false,
    success: function(data, textStatus, jqXHR)
    {
      $("#notifications").removeClass('is-danger').addClass('is-success');
      $("#notifications-content").html(data.message)
      $("#notifications").fadeIn("slow")
    },
    error: function(jqXHR, textStatus, errorThrown)
    {
      $("#notifications").removeClass('is-success').addClass('is-danger');
      $("#notifications-content").html(jqXHR.responseJSON.error)
      $("#notifications").fadeIn("slow")
    }
  });
}
