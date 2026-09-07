import 'package:shared_preferences/shared_preferences.dart';

class SettingsStore {
  static const _baseUrlKey = 'gateway_base_url';
  static const _tokenKey = 'gateway_token';

  Future<GatewaySettings> load() async {
    final preferences = await SharedPreferences.getInstance();
    return GatewaySettings(
      baseUrl: preferences.getString(_baseUrlKey) ?? '',
      token: preferences.getString(_tokenKey) ?? '',
    );
  }

  Future<void> save(GatewaySettings settings) async {
    final preferences = await SharedPreferences.getInstance();
    await preferences.setString(_baseUrlKey, settings.baseUrl.trim());
    await preferences.setString(_tokenKey, settings.token);
  }
}

class GatewaySettings {
  const GatewaySettings({required this.baseUrl, required this.token});

  final String baseUrl;
  final String token;
  bool get configured => baseUrl.isNotEmpty && token.isNotEmpty;
}
