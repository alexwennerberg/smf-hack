<?php
// Mint an SMF persistent-login cookie value for a member id (parity tooling).
require(dirname(__FILE__) . '/Settings.php');
$c = mysql_connect($db_server, $db_user, $db_passwd); mysql_select_db($db_name, $c);
$id = (int) $_GET['id'];
$r = mysql_query("SELECT passwd, passwordSalt FROM {$db_prefix}members WHERE ID_MEMBER = $id");
$row = mysql_fetch_assoc($r);
$hash = sha1($row['passwd'] . $row['passwordSalt']);
echo serialize(array($id, $hash, time() + 86400, 0));
