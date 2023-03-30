package cmd

const formtemplate = `
<!DOCTYPE html>
<!-- -*-web-*- -->
<html lang="en">
  <head>
    <meta charset="UTF-8">
    <meta http-equiv="Content-Type" content="text/html; charset=UTF-8" />
    <meta http-equiv="X-UA-Compatible" content="IE=edge" />
    <meta name="description" content="upload form" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>File upload form</title>
    <script src="https://ajax.googleapis.com/ajax/libs/jquery/3.5.1/jquery.min.js"></script>
    <link href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.0-alpha2/dist/css/bootstrap.min.css"
          rel="stylesheet" integrity="sha384-aFq/bzH65dt+w6FI2ooMVUpc+21e0SRygnTpmBvdBgSdnuTN7QbdgL+OapgHtvPp" crossorigin="anonymous">
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/bootstrap-icons@1.5.0/font/bootstrap-icons.css">
  </head>
  <body>
    <div class="container">
    <h4>Upload form {{ .Id }}</h4>
    <!-- Response -->
    <div class="statusMsg"></div>

    <!-- File upload form -->
    <div class="col-lg-12">
      <form id="UploadForm" enctype="multipart/form-data" action="/v1/uploads" method="POST">
        <div class="mb-3 row">
          <p>
            Use this form to upload one or more files. The creator of the form will automatically get notified.
          </p>
        </div>
        <div class="mb-3 row">
          <label class="col-sm-2 col-form-label">Description</label>
          <label class="col-sm-10 col-form-label">{{ .Description}} </label>
        </div>
        <div class="mb-3 row">
          <label for="file" class="col-sm-2 col-form-label">Select</label>
          <div class="col-sm-10">
            <input type="file" class="form-control" id="file" name="uploads[]" multiple
                   />
          </div>
        </div>

        <div class="mb-3 row">
          <label for="display" class="col-sm-2 col-form-label">Selected Files</label>
          <div class="col-sm-10">
            <!-- <input type="textara" class="form-control" id="upload-file-info" readonly>-->
            <div id="upload-file-info"></div>
          </div>
        </div>
        
        <input type="submit" name="submit" class="btn btn-success submitBtn" value="Upload"/>
      </form>
    </div>
    </div>
    <script src="https://cdn.jsdelivr.net/npm/bootstrap@5.3.0-alpha2/dist/js/bootstrap.bundle.min.js"
            integrity="sha384-qKXV1j0HvMUeCBQ+QVp7JcfGl760yU08IQ+GpUo5hlbpg51QRiuqHAJz8+BrxE/N" crossorigin="anonymous"></script>
    <script>
     $(document).ready(function(){
       // Submit form data via Ajax
       $("#UploadForm").on('submit', function(e){
         e.preventDefault();
         $.ajax({
           type: 'POST',
           url: '/v1/uploads',
           data: new FormData(this),
           dataType: 'json',
           contentType: false,
           cache: false,
           processData:false,
           beforeSend: function(xhr){
               $('.submitBtn').attr("disabled","disabled");
               $('#UploadForm').css("opacity",".5");
               xhr.setRequestHeader('Authorization', 'Bearer {{.Id}}');
           },
           success: function(response){
             $('.statusMsg').html('');
             if(response.success){
                 $('#UploadForm')[0].reset();
                 $('.statusMsg').html('<p class="alert alert-success">Your upload is available for download.<!-- '
                                      +response.uploads[0].url+' -->');
                 $('#UploadForm').hide();
             }else{
               $('.statusMsg').html('<p class="alert alert-danger">'+response.message+'</p>');
             }
             $('#UploadForm').css("opacity","");
             $(".submitBtn").removeAttr("disabled");
           }
         });
       });

       $("#file").on('change', function() {
         $("#upload-file-info").empty();
         for (var i = 0; i < $(this).get(0).files.length; ++i) {
           $("#upload-file-info").append('<i class="bi-check-lg"></i> ' + $(this).get(0).files[i].name + '<br>');
         }
       });
     });
    </script>

  </body>
</html>

    
`
