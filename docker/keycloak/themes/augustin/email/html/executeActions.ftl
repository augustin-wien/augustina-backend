<#-- Admin-triggered account mail (backoffice buttons, welcome mail for new vendors
     and abo customers). Keycloak uses this one template for every action, so pick a
     text that says what the link does instead of listing the action names. -->
<#import "template.ftl" as layout>
<#assign actions = requiredActions![]>
<#if actions?size == 1 && actions[0] == "VERIFY_EMAIL">
  <#assign kind = "verifyEmail">
<#elseif actions?seq_contains("UPDATE_PASSWORD")>
  <#assign kind = "updatePassword">
<#else>
  <#assign kind = "executeActions">
</#if>
<#outputformat "plainText">
<#assign requiredActionsText><#list actions as a>${msg("requiredAction.${a}")}<#sep>, </#sep></#list></#assign>
</#outputformat>
<#-- Customers created from an order get their email as first name -->
<#assign firstName = (user.firstName)!"">
<@layout.emailLayout>
<div style="font-family: Arial, Helvetica, sans-serif; font-size: 16px; line-height: 1.5; color: #222222; max-width: 560px;">
  <p><#if firstName?has_content && !firstName?contains("@")>${msg("accountEmailGreetingName", firstName)}<#else>${msg("accountEmailGreeting")}</#if></p>
  <p>${msg(kind + "EmailIntro", requiredActionsText)}</p>
  <p style="margin: 28px 0;">
    <a href="${link}" style="display: inline-block; background: #d74766; color: #ffffff; text-decoration: none; font-weight: bold; padding: 12px 24px; border-radius: 4px;">${msg(kind + "EmailButton")}</a>
  </p>
  <#if kind == "updatePassword"><p>${msg("updatePasswordEmailAfter")}</p></#if>
  <p>${msg("accountEmailLinkExpiry", linkExpirationFormatter(linkExpiration))}</p>
  <p style="color: #666666; font-size: 14px;">${msg("accountEmailButtonFallback")}<br><a href="${link}" style="color: #666666; word-break: break-all;">${link}</a></p>
  <p style="color: #666666; font-size: 14px;">${msg("accountEmailIgnore")}</p>
  <p>${msg("accountEmailRegards")}<br>${msg("accountEmailSignature")}</p>
</div>
</@layout.emailLayout>
