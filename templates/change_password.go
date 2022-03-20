package templates

var HtmlPasswordResetTemplateHead = `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<html xmlns="http://www.w3.org/1999/xhtml">
<head>
</head>
<body>
Click on the given link to reset your password <a href="https://dalal.pragyan.org/changepassword"> https://dalal.pragyan.org/changepassword </a> .
Your temporary password is
`

var HtmlPasswordResetTemplateTail = `</body>
</html>
`

var PlainPasswordResetTemplate = `Please reset your password at https://dalal.pragyan.org/resetPassword/. Your temporary password is %s`
