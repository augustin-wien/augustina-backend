<#import "template.ftl" as layout>
<@layout.registrationLayout displayMessage=false; section>
    <#if section = "header">
        ${msg("termsTitle")}
    <#elseif section = "form">
    <div id="kc-terms-text">
        ${kcSanitize(msg("termsText"))?no_esc}
        <p><a href="https://augustin.or.at/datenschutzerklaerung/" target="_blank">https://augustin.or.at/datenschutzerklaerung/</a></p>
    </div>
    <form class="form-actions" action="${url.loginAction}" method="POST">
        <input class="${properties.kcButtonClass!} ${properties.kcButtonPrimaryClass!} ${properties.kcButtonLargeClass!}" name="accept" id="kc-accept" type="submit" value="${msg("doAccept")}"/>
    </form>
    <div class="clearfix"></div>
    </#if>
</@layout.registrationLayout>
