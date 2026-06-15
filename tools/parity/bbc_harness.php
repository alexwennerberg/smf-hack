<?php
// Parity harness: run SMF's real parse_bbc() over a corpus. POST a JSON array
// of base64-encoded inputs; returns a JSON array of base64-encoded outputs.
$ssi_maintenance_off = true;
ob_start();
require(dirname(__FILE__) . '/SSI.php');
ob_end_clean();
header('Content-Type: application/json');
$in = json_decode(file_get_contents('php://input'), true);
$out = array();
if (is_array($in)) {
    foreach ($in as $b64) {
        $text = base64_decode($b64);
        $out[] = base64_encode(parse_bbc($text, true));
    }
}
echo json_encode($out);
exit;
