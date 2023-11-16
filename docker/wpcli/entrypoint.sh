# /bin/sh
chown xfs:xfs /wpcli/parser -R &&

cd /var/www/html &&
rm -rf /var/www/html/wp-content/uploads/* &&
echo "Waiting for database..." &&
while ! nc -z wordpress-db 3306; do
  sleep 0.1
done &&
echo "Database started" &&
echo "Set up WordPress" &&
wp --allow-root config create --dbname=wordpress --dbuser=wordpress --dbpass=wordpress --dbhost=wordpress-db --force &&
echo "Drop database" &&
(wp --allow-root db drop --yes) || true &&
echo "Create database" &&
wp --allow-root db create || true &&
echo "Install WordPress" &&
wp --allow-root core install --url=localhost:8090 --title="Augustin" --admin_user=test_superuser --admin_password=Test123! --admin_email=test_superuser@example.com &&
echo "Install plugins and themes" &&
wp --allow-root theme activate augustin-wp-theme &&
wp --allow-root config set OIDC_LOGIN_TYPE auto &&
wp --allow-root config set OIDC_CLIENT_ID wordpress &&
wp --allow-root config set OIDC_CLIENT_SECRET 84uZmW6FlEPgvUd201QUsWRmHzUIamZB &&
wp --allow-root config set OIDC_ENDPOINT_LOGIN_URL http://keycloak:8080/realms/augustin/protocol/openid-connect/auth &&
wp --allow-root config set OIDC_ENDPOINT_USERINFO_URL http://keycloak:8080/realms/augustin/protocol/openid-connect/userinfo &&
wp --allow-root config set OIDC_ENDPOINT_TOKEN_URL http://keycloak:8080/realms/augustin/protocol/openid-connect/token &&
wp --allow-root config set OIDC_ENDPOINT_LOGOUT_URL http://keycloak:8080/realms/augustin/protocol/openid-connect/logout &&
wp --allow-root config set OIDC_CLIENT_SCOPE "email profile openid offline_access roles" &&
wp --allow-root config set OIDC_ENFORCE_PRIVACY true --raw &&
wp --allow-root config set OIDC_CREATE_IF_DOES_NOT_EXIST true --raw &&
wp --allow-root config set OIDC_LINK_EXISTING_USERS true --raw &&
wp --allow-root config set OIDC_REDIRECT_USER_BACK false --raw &&
wp --allow-root config set OIDC_REDIRECT_ON_LOGOUT false --raw &&
wp --allow-root config set WP_ENVIRONMENT_TYPE local &&
wp --allow-root config set WP_DEBUG true --raw &&
wp --allow-root plugin install --activate --force groups &&
wp --allow-root plugin install --activate --force daggerhart-openid-connect-generic &&
wp --allow-root plugin install --activate --force create-block-theme &&
wp --allow-root plugin activate augustin-wp-papers &&
wp --allow-root rewrite structure '/%year%/%monthnum%/%postname%/' --hard &&
wp --allow-root term create category_papers newspaper &&
wp --allow-root term create category_papers magazin-1 --parent="2" &&
wp --allow-root term create category_papers magazin-2 --parent="2" &&
wp --allow-root term create category_papers magazin-3 --parent="2" &&
wp --allow-root user create author1 author1@example.com --role="author"  --display_name="author1" &&
wp --allow-root user create author2 author2@example.com --role="author"  --display_name="author2" &&
wp --allow-root user create author3 author3@example.com --role="author"  --display_name="author3" &&
wp --allow-root post create --post_title="magazin-1" --post_status=publish --post_type="papers" /wpcli/demo_content/papers/paper-1/magazin-1.txt &&
wp --allow-root post create --post_title="test2" --post_status=publish --post_type="papers" /wpcli/demo_content/papers/paper-2/magazin-2.txt &&
wp --allow-root post create --post_title="test3" --post_status=publish --post_type="papers" /wpcli/demo_content/papers/paper-3/magazin-3.txt &&
wp --allow-root post term add 5 category_papers magazin-1 &&
wp --allow-root post term add 6 category_papers magazin-2 &&
wp --allow-root post term add 7 category_papers magazin-3 &&
wp --allow-root term create category einsicht &&
wp --allow-root term create category "augustiner:in" &&
wp --allow-root term create category "das wahre leben" &&
wp --allow-root term create category "cover" &&
wp --allow-root term create category "tun & lassen" &&
wp --allow-root term create category "vorstadt" &&
wp --allow-root term create category "lokalmatador:in nº" &&
wp --allow-root term create category "art.ist.in" &&
wp --allow-root term create category "dichter innenteil" &&
wp --allow-root term create category "augustinchen" &&
wp --allow-root media import /wpcli/demo_content/papers/paper-1/magazin-1.png --post_id=5 --title="Magazin 1" --featured_image &&
wp --allow-root media import /wpcli/demo_content/papers/paper-2/magazin-2.jpeg --post_id=6--title="Magazin 2" --featured_image &&
wp --allow-root media import /wpcli/demo_content/papers/paper-3/magazin-3.png --post_id=7 --title="Magazin 3" --featured_image &&
wp --allow-root post create --post_title="Konflikte für den Weltfrieden" --post_status=publish --post_author="2" --post_type="articles" /wpcli/demo_content/papers/paper-1/article-1.txt &&
wp --allow-root post create --post_title="«Rätsel» lösen mit Bildern" --post_status=publish --post_author="3" --post_type="articles" /wpcli/demo_content/papers/paper-1/article-2.txt &&
wp --allow-root post create --post_title="Gierflation" --post_status=publish --post_author="4" --post_type="articles" /wpcli/demo_content/papers/paper-1/article-3.txt &&
wp --allow-root post term add 11 category_papers magazin-1 &&
wp --allow-root post term add 11 category editorial &&
wp --allow-root post term add 12 category_papers magazin-1 &&
wp --allow-root post term add 12 category augustinerin &&
wp --allow-root post term add 13 category_papers magazin-1 &&
wp --allow-root post term add 13 category das-wahre-leben &&
wp --allow-root post create --post_title="K-Wörter" --post_status=publish --post_author="4" --post_type="articles" /wpcli/demo_content/papers/paper-2/article-1.txt &&
wp --allow-root post create --post_title="Vom Glück, eine Arbeit zu haben" --post_status=publish --post_author="2" --post_type="articles" /wpcli/demo_content/papers/paper-2/article-2.txt &&
wp --allow-root post create --post_title="Versteinerung der Kinder-und Jugendhilfe" --post_status=publish --post_author="3" --post_type="articles" /wpcli/demo_content/papers/paper-2/article-3.txt &&
wp --allow-root post term add 14 category_papers magazin-2 &&
wp --allow-root post term add 14 category editorial &&
wp --allow-root post term add 15 category_papers magazin-2 &&
wp --allow-root post term add 15 category augustinerin &&
wp --allow-root post term add 16 category_papers magazin-2 &&
wp --allow-root post term add 16 category das-wahre-leben &&
wp --allow-root post create --post_title="Setzen!" --post_status=publish --post_author="3" --post_type="articles" /wpcli/demo_content/papers/paper-3/article-1.txt &&
wp --allow-root post create --post_title="«Schreiben war immer schon mein Ding» lösen mit Bildern" --post_status=publish --post_author="4 " --post_type="articles" /wpcli/demo_content/papers/paper-3/article-2.txt &&
wp --allow-root post create --post_title="Einstürzende Bedürfnispyramiden" --post_status=publish --post_author="2" --post_type="articles" /wpcli/demo_content/papers/paper-3/article-3.txt &&
wp --allow-root post term add 17 category_papers magazin-3 &&
wp --allow-root post term add 17 category editorial &&
wp --allow-root post term add 18 category_papers magazin-3 &&
wp --allow-root post term add 18 category augustinerin &&
wp --allow-root post term add 19 category_papers magazin-3 &&
wp --allow-root post term add 19 category das-wahre-leben &&
wp --allow-root media import /wpcli/demo_content/papers/paper-1/article-1.jpg --post_id=11 --title="Article 1" --featured_image &&
wp --allow-root media import /wpcli/demo_content/papers/paper-1/article-2.jpg --post_id=12 --title="Article 2" --featured_image &&
wp --allow-root media import /wpcli/demo_content/papers/paper-1/article-3.jpg --post_id=13 --title="Article 3" --featured_image &&
wp --allow-root media import /wpcli/demo_content/papers/paper-2/article-1.jpg --post_id=14 --title="Article 4" --featured_image &&
wp --allow-root media import /wpcli/demo_content/papers/paper-2/article-2.jpg --post_id=15 --title="Article 5" --featured_image &&
wp --allow-root media import /wpcli/demo_content/papers/paper-2/article-3.jpg --post_id=16 --title="Article 6" --featured_image &&
wp --allow-root media import /wpcli/demo_content/papers/paper-3/article-1.jpg --post_id=17 --title="Article 7" --featured_image &&
wp --allow-root media import /wpcli/demo_content/papers/paper-3/article-2.jpg --post_id=18 --title="Article 8" --featured_image &&
wp --allow-root media import /wpcli/demo_content/papers/paper-3/article-3.jpg --post_id=19 --title="Article 9" --featured_image &&
wp --allow-root user create api_user user@user.com --role="administrator" --user_pass="Test123!" &&
wp --allow-root user application-password delete api_user --all  --allow-root &&
# echo WP_API_KEY=$(wp --allow-root user application-password create api_user augustin --porcelain --allow-root)>/wpcli/parser/.env &&
wp --allow-root media import "/wpcli/demo_content/logo.png" --porcelain | wp --allow-root option update site_icon &&
wp --allow-root media import "/wpcli/demo_content/logo.png" --porcelain | wp --allow-root option update site_logo  &&
# wp --allow-root menu create "Top Menu" &&
# wp --allow-root menu item add-custom top-menu Ausgaben / &&
# wp --allow-root menu location assign top-menu primary
echo "Done"&&bash