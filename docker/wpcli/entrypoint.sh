# /bin/sh
mkdir -p /etc/X11/fs && chown xfs:xfs /etc/X11/fs -R &&
su -l xfs -s /bin/sh -c '

      cd /var/www/html &&
      rm -rf /var/www/html/wp-content/uploads/* &&
      wp config create --dbname=wordpress --dbuser=wordpress --dbpass=wordpress --dbhost=wordpress-db --force &&
      wp db drop --yes || true &&
      wp db create || true &&
      wp core install --url=localhost:8090 --title="Augustin" --admin_user=test_superuser --admin_password=Test123! --admin_email=test_superuser@example.com &&
      wp theme activate augustin-wp-theme &&
      wp config set OIDC_LOGIN_TYPE auto &&
      wp config set OIDC_CLIENT_ID wordpress &&
      wp config set OIDC_CLIENT_SECRET 84uZmW6FlEPgvUd201QUsWRmHzUIamZB &&
      wp config set OIDC_ENDPOINT_LOGIN_URL http://keycloak:8080/realms/augustin/protocol/openid-connect/auth &&
      wp config set OIDC_ENDPOINT_USERINFO_URL http://keycloak:8080/realms/augustin/protocol/openid-connect/userinfo &&
      wp config set OIDC_ENDPOINT_TOKEN_URL http://keycloak:8080/realms/augustin/protocol/openid-connect/token &&
      wp config set OIDC_ENDPOINT_LOGOUT_URL http://keycloak:8080/realms/augustin/protocol/openid-connect/logout &&
      wp config set OIDC_CLIENT_SCOPE "email profile openid offline_access roles" &&
      wp config set OIDC_ENFORCE_PRIVACY true --raw &&
      wp config set OIDC_CREATE_IF_DOES_NOT_EXIST true --raw &&
      wp config set OIDC_LINK_EXISTING_USERS true --raw &&
      wp config set OIDC_REDIRECT_USER_BACK false --raw &&
      wp config set OIDC_REDIRECT_ON_LOGOUT false --raw &&
      wp config set WP_ENVIRONMENT_TYPE development &&
      wp config set WP_DEBUG true --raw &&
      wp plugin install --activate --force groups &&
      wp plugin install --activate --force daggerhart-openid-connect-generic &&
      wp plugin activate augustin-wp-papers &&
      wp rewrite structure '/%year%/%monthnum%/%postname%/' --hard &&
      wp term create category_papers newspaper && 
      wp term create category_papers magazin-1 --parent="2" &&
      wp term create category_papers magazin-2 --parent="2" &&
      wp term create category_papers magazin-3 --parent="2" &&
      wp post create --post_title="test1" --post_status=publish --post_content="1 was nettes" --post_type="papers" &&
      wp post create --post_title="test2" --post_status=publish --post_content="2 was nettes" --post_type="papers" &&
      wp post create --post_title="test3" --post_status=publish --post_content="3 was nettes" --post_type="papers" &&
      wp post term add 4 category_papers magazin-1 && 
      wp post term add 5 category_papers magazin-2 && 
      wp post term add 6 category_papers magazin-3 && 
      wp media import /wpcli/demo_content/startpages/magazin-1.png --post_id=4 --title="Magazin 1" --featured_image &&
      wp media import /wpcli/demo_content/startpages/magazin-2.jpeg --post_id=5 --title="Magazin 2" --featured_image &&
      wp media import /wpcli/demo_content/startpages/magazin-3.png --post_id=6 --title="Magazin 3" --featured_image &&
      wp post create --post_title="test article 1" --post_status=publish --post_content="1 was nettes" --post_type="articles" &&
      wp post create --post_title="test article 2" --post_status=publish --post_content="2 was nettes" --post_type="articles" &&
      wp post create --post_title="test article 3" --post_status=publish --post_content="3 was nettes" --post_type="articles" &&
      wp post term add 10 category_papers magazin-1 && 
      wp post term add 11 category_papers magazin-2 && 
      wp post term add 12 category_papers magazin-3 &&
      wp term create category einsicht &&
      wp term create category "augustiner:in" &&
      wp term create category "das wahre leben" &&
      wp term create category "cover" &&
      wp term create category "tun & lassen" &&
      wp term create category "vorstadt" &&
      wp term create category "lokalmatador:in nº 520" &&
      wp term create category "art.ist.in" &&
      wp term create category "dichter innenteil" &&
      wp term create category "augustinchen" &&
      wp user create api_user user@user.com --role="administrator" --user_pass="Test123!" &&
      wp user application-password delete api_user --all  --allow-root &&  
      echo WP_API_KEY=$(wp user application-password create api_user augustin --porcelain --allow-root)>/wpcli/.env.parser
'