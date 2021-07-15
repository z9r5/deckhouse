$( document ).ready(function() {
    if ($.cookie("demotoken") || $.cookie("license-token") ) {
        let username = 'license-token';
        let password = $.cookie("license-token") ? $.cookie("license-token") : $.cookie("demotoken");
        let registry = 'registry.deckhouse.io';
        let auth = btoa(username + ':' + password);
        let config = '{"auths": { "'+ registry +'": { "username": "'+ username +'", "password": "' + password + '", "auth": "' + auth +'"}}}';
        let matchStringClusterConfig = '<YOUR_ACCESS_STRING_IS_HERE>';
        let matchStringDockerLogin = "<LICENSE_TOKEN>";

        $('code span.s').filter(function () {
            return this.innerText == matchStringClusterConfig;
        }).text(btoa(config));

        $('.highlight code').filter(function () {
            return this.innerText.match(matchStringDockerLogin) == matchStringDockerLogin;
        }).each(function(index) {
            $(this).text($(this).text().replace(matchStringDockerLogin,password));
        });
    } else {
        console.log("No license token, so InitConfiguration was not updated");
    }
});



$(document).ready(function() {
    $('[gs-revision-tabs]').on('click', function() {
        var name = $(this).attr('data-features-tabs-trigger');
        var $parent = $(this).closest('[data-features-tabs]');
        var $triggers = $parent.find('[data-features-tabs-trigger]');
        var $contents = $parent.find('[data-features-tabs-content]');
        var $content = $parent.find('[data-features-tabs-content=' + name + ']');

        $triggers.removeClass('active');
        $contents.removeClass('active');

        $(this).addClass('active');
        $content.addClass('active');
    })
});
