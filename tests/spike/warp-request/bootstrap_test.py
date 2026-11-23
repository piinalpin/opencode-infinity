import base64
import copy
import unittest
import bootstrap


class BootstrapTests(unittest.TestCase):
    def sample(self):
        return {
            "id": "12345678-1234-1234-1234-123456789abc",
            "token": "synthetic-token-only",
            "config": {
                "interface": {"addresses": {"v4": "172.16.0.2"}},
                "peers": [{"public_key": base64.b64encode(bytes([1]) * 32).decode(),
                           "endpoint": {"host": "engage.cloudflareclient.com:2408"}}],
            },
        }

    def test_profile_validated_and_pinned(self):
        private = base64.b64encode(bytes([2]) * 32).decode()
        profile = bootstrap.make_profile(private, self.sample(), lambda host: "162.159.192.1")
        self.assertIsInstance(profile, str, "private profile belum dibentuk")
        self.assertIn("Endpoint = 162.159.192.1:2408", profile)
        self.assertIn("Address = 172.16.0.2/32", profile)
        self.assertNotIn("token", profile)
        for kind in ["private", "peer", "endpoint", "address", "token", "id"]:
            with self.subTest(kind=kind):
                data = copy.deepcopy(self.sample())
                key = private
                if kind == "private": key = "bad\nPrivateKey=INJECT"
                if kind == "peer": data["config"]["peers"][0]["public_key"] = "bad"
                if kind == "endpoint": data["config"]["peers"][0]["endpoint"]["host"] = "evil.invalid:80"
                if kind == "address": data["config"]["interface"]["addresses"]["v4"] = "bad; command"
                if kind == "token": data["token"] = "secret\r\nHeader: injected"
                if kind == "id": data["id"] = "../../arbitrary"
                with self.assertRaises(bootstrap.BootstrapError) as failure:
                    bootstrap.make_profile(key, data, lambda host: "162.159.192.1")
                self.assertNotIn("secret", str(failure.exception))

    def test_single_registration_and_enable_no_retry(self):
        calls = []
        def send(method, url, data, headers):
            calls.append((method, url, data, headers))
            return self.sample() if method == "POST" else {}
        private = base64.b64encode(bytes([2]) * 32).decode()
        public = base64.b64encode(bytes([3]) * 32).decode()
        profile = bootstrap.register_profile(private, public, send, lambda host: "162.159.192.1")
        self.assertEqual([c[0] for c in calls], ["POST", "PATCH"], "registrasi/enable belum dieksekusi tepat sekali")
        self.assertEqual(calls[0][2]["key"], public)
        self.assertNotIn(private, repr(calls))
        self.assertIsInstance(profile, str)
        failures = []
        def fail(*args):
            failures.append(1)
            raise OSError("PRIVATE-TOKEN-NOT-FOR-OUTPUT")
        with self.assertRaises(bootstrap.BootstrapError) as caught:
            bootstrap.register_profile(private, public, fail, lambda host: "162.159.192.1")
        self.assertEqual(len(failures), 1)
        self.assertNotIn("PRIVATE", str(caught.exception))


if __name__ == "__main__":
    unittest.main()
