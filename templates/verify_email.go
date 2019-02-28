package templates

var HtmlEmailVerificationTemplate = `
<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<html xmlns="http://www.w3.org/1999/xhtml">
<head>
<style>
.content{
	margin:10%;
}
.heading{
	color:#555555;
}
.subheading{
	color:#9e9e9e;
}
</style>
</head>
<body>
<div class="content">
	<h2>Dalal Street</h2>
	<p class="heading"><strong>You are one click away from your account on Dalal Street.</strong></p>
    <p class="subheading">Follow this link to create your account.</p>
    %s
</div>
</body>
</html>
`
var PlainEmailVerificationTemplate = `Please verify your account at %s`
