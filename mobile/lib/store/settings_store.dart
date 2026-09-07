import 'package:flutter/foundation.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../api/gateway_client.dart';

class GatewaySettings {
  const GatewaySettings({required this.baseUrl, required this.token});
  final String baseUrl, token;
  bool get configured => baseUrl.isNotEmpty && token.isNotEmpty;
}

class SettingsStore extends ChangeNotifier {
  static const _baseUrlKey = 'gateway_base_url', _tokenKey = 'gateway_token';
  SettingsStore(this._settings);
  GatewaySettings _settings;
  GatewaySettings get settings => _settings;
  GatewayClient client() =>
      GatewayClient(baseUrl: _settings.baseUrl, token: _settings.token);
  static Future<SettingsStore> load() async {
    final p = await SharedPreferences.getInstance();
    return SettingsStore(
      GatewaySettings(
        baseUrl: p.getString(_baseUrlKey) ?? '',
        token: p.getString(_tokenKey) ?? '',
      ),
    );
  }

  Future<void> save(GatewaySettings value) async {
    final p = await SharedPreferences.getInstance();
    _settings = GatewaySettings(
      baseUrl: value.baseUrl.trim(),
      token: value.token,
    );
    await p.setString(_baseUrlKey, _settings.baseUrl);
    await p.setString(_tokenKey, _settings.token);
    notifyListeners();
  }
}
