import 'package:flutter/foundation.dart';
import '../models/llm_config.dart';
import '../services/api.dart';

class SettingsProvider extends ChangeNotifier {
  final ApiService _api = ApiService();

  LLMConfig? _config;
  bool _isLoading = false;
  String? _error;

  LLMConfig? get config => _config;
  bool get isLoading => _isLoading;
  String? get error => _error;

  Future<void> loadConfig() async {
    _isLoading = true;
    _error = null;
    notifyListeners();

    try {
      _config = await _api.getLLMConfig();
      _isLoading = false;
      notifyListeners();
    } catch (e) {
      _error = e.toString();
      _isLoading = false;
      notifyListeners();
    }
  }

  Future<void> updateConfig(LLMConfig config) async {
    _isLoading = true;
    _error = null;
    notifyListeners();

    try {
      _config = await _api.updateLLMConfig(config);
      _isLoading = false;
      notifyListeners();
    } catch (e) {
      _error = e.toString();
      _isLoading = false;
      notifyListeners();
    }
  }

  void clearError() {
    _error = null;
    notifyListeners();
  }
}