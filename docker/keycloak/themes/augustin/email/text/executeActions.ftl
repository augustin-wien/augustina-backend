<#ftl output_format="plainText">
<#-- Plain text twin of html/executeActions.ftl -->
<#assign actions = requiredActions![]>
<#if actions?size == 1 && actions[0] == "VERIFY_EMAIL">
  <#assign kind = "verifyEmail">
<#elseif actions?seq_contains("UPDATE_PASSWORD")>
  <#assign kind = "updatePassword">
<#else>
  <#assign kind = "executeActions">
</#if>
<#assign requiredActionsText><#list actions as a>${msg("requiredAction.${a}")}<#sep>, </#sep></#list></#assign>
<#assign firstName = (user.firstName)!"">
<#if firstName?has_content && !firstName?contains("@")>${msg("accountEmailGreetingName", firstName)}<#else>${msg("accountEmailGreeting")}</#if>

${msg(kind + "EmailIntro", requiredActionsText)}

${msg(kind + "EmailButton")}:
${link}

<#if kind == "updatePassword">
${msg("updatePasswordEmailAfter")}

</#if>
${msg("accountEmailLinkExpiry", linkExpirationFormatter(linkExpiration))}

${msg("accountEmailIgnore")}

${msg("accountEmailRegards")}
${msg("accountEmailSignature")}
