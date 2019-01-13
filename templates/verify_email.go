package templates

var HtmlEmailVerificationTemplate = `
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<html xmlns="http://www.w3.org/1999/xhtml">
<head>
</head>
<body>
  Thanks for registring on dalalstreet. Here is your url %s
</body>
</html>
`
var PlainEmailVerificationTemplate = `Please verify your account at %s`
