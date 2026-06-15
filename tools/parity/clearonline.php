<?php
require(dirname(__FILE__) . '/Settings.php');
$c = mysql_connect($db_server, $db_user, $db_passwd); mysql_select_db($db_name, $c);
mysql_query("DELETE FROM {$db_prefix}log_online");
echo "ok";
