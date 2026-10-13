package at.augustin.keycloak.invite;

import com.fasterxml.jackson.annotation.JsonProperty;
import org.keycloak.authentication.actiontoken.DefaultActionToken;

/**
 * Signs a reader in from an invite link – in the browser, on Keycloak's own
 * host, so the session cookie and a password set there belong to the login
 * domain (password managers file the password under it).
 */
public class InviteActionToken extends DefaultActionToken {

    public static final String TOKEN_TYPE = "augustin-invite";

    @JsonProperty("rdu")
    private String redirectUri;

    @JsonProperty("upw")
    private boolean updatePassword;

    public InviteActionToken(String userId, int absoluteExpirationInSecs, String clientId,
                             String redirectUri, boolean updatePassword) {
        super(userId, TOKEN_TYPE, absoluteExpirationInSecs, null);
        this.issuedFor = clientId;
        this.redirectUri = redirectUri;
        this.updatePassword = updatePassword;
    }

    private InviteActionToken() {
        // Jackson
    }

    public String getRedirectUri() {
        return redirectUri;
    }

    public boolean isUpdatePassword() {
        return updatePassword;
    }
}
