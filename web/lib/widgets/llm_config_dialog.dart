import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../models/llm_config.dart';
import '../providers/settings_provider.dart';

class LLMConfigDialog extends StatefulWidget {
  const LLMConfigDialog({super.key});

  @override
  State<LLMConfigDialog> createState() => _LLMConfigDialogState();
}

class _LLMConfigDialogState extends State<LLMConfigDialog> {
  final TextEditingController _apiKeyController = TextEditingController();
  final TextEditingController _baseUrlController = TextEditingController();
  final TextEditingController _modelController = TextEditingController();
  bool _obscureApiKey = true;
  bool _isLoading = false;
  bool _initialized = false;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      context.read<SettingsProvider>().loadConfig();
    });
  }

  @override
  void dispose() {
    _apiKeyController.dispose();
    _baseUrlController.dispose();
    _modelController.dispose();
    super.dispose();
  }

  void _initControllers(LLMConfig config) {
    _apiKeyController.text = '';
    _baseUrlController.text = config.baseUrl;
    _modelController.text = config.model;
    _initialized = true;
  }

  Future<void> _handleSave(SettingsProvider provider) async {
    if (_isLoading) return;

    setState(() {
      _isLoading = true;
    });

    final apiKey = _apiKeyController.text.trim();
    final baseUrl = _baseUrlController.text.trim();
    final model = _modelController.text.trim();

    final newConfig = LLMConfig(
      apiKey: apiKey,
      baseUrl: baseUrl,
      model: model,
      hasApiKey: apiKey.isNotEmpty,
      updatedAt: '',
    );

    try {
      await provider.updateConfig(newConfig);
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('配置已保存')),
        );
        Navigator.pop(context);
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('保存失败: $e')),
        );
      }
    } finally {
      if (mounted) {
        setState(() {
          _isLoading = false;
        });
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    return Consumer<SettingsProvider>(
      builder: (context, provider, _) {
        if (provider.isLoading && provider.config == null) {
          return const AlertDialog(
            title: Text('LLM Configuration'),
            content: SizedBox(
              width: 400,
              height: 200,
              child: Center(child: CircularProgressIndicator()),
            ),
          );
        }

        final config = provider.config;
        if (config != null && !_initialized) {
          _initControllers(config);
        }

        return AlertDialog(
          title: Row(
            children: [
              Container(
                padding: const EdgeInsets.all(8),
                decoration: BoxDecoration(
                  color: const Color(0xFF8B5CF6).withAlpha(38),
                  borderRadius: BorderRadius.circular(8),
                ),
                child: const Icon(
                  Icons.settings,
                  color: Color(0xFF8B5CF6),
                  size: 24,
                ),
              ),
              const SizedBox(width: 12),
              const Text('LLM Configuration'),
            ],
          ),
          content: SizedBox(
            width: 400,
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                const Text('API Key', style: TextStyle(fontWeight: FontWeight.w500)),
                const SizedBox(height: 8),
                TextField(
                  controller: _apiKeyController,
                  obscureText: _obscureApiKey,
                  decoration: InputDecoration(
                    hintText: config?.hasApiKey == true ? config!.apiKey : 'Enter API Key',
                    suffixIcon: IconButton(
                      icon: Icon(_obscureApiKey ? Icons.visibility : Icons.visibility_off),
                      onPressed: () {
                        setState(() {
                          _obscureApiKey = !_obscureApiKey;
                        });
                      },
                    ),
                    border: OutlineInputBorder(
                      borderRadius: BorderRadius.circular(8),
                    ),
                    contentPadding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
                  ),
                ),
                const SizedBox(height: 16),
                const Text('Base URL', style: TextStyle(fontWeight: FontWeight.w500)),
                const SizedBox(height: 8),
                TextField(
                  controller: _baseUrlController,
                  decoration: InputDecoration(
                    hintText: 'https://coding.dashscope.aliyuncs.com/v1',
                    border: OutlineInputBorder(
                      borderRadius: BorderRadius.circular(8),
                    ),
                    contentPadding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
                  ),
                ),
                const SizedBox(height: 16),
                const Text('Model', style: TextStyle(fontWeight: FontWeight.w500)),
                const SizedBox(height: 8),
                TextField(
                  controller: _modelController,
                  decoration: InputDecoration(
                    hintText: 'glm-5',
                    border: OutlineInputBorder(
                      borderRadius: BorderRadius.circular(8),
                    ),
                    contentPadding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
                  ),
                ),
                if (config != null) ...[
                  const SizedBox(height: 16),
                  Row(
                    children: [
                      Icon(
                        config.hasApiKey ? Icons.check_circle : Icons.warning,
                        color: config.hasApiKey ? Colors.green : Colors.orange,
                        size: 16,
                      ),
                      const SizedBox(width: 8),
                      Text(
                        config.hasApiKey ? 'API Key configured' : 'API Key not configured',
                        style: TextStyle(
                          color: config.hasApiKey ? Colors.green : Colors.orange,
                          fontSize: 12,
                        ),
                      ),
                    ],
                  ),
                ],
              ],
            ),
          ),
          actions: [
            TextButton(
              onPressed: _isLoading ? null : () => Navigator.pop(context),
              child: const Text('Cancel'),
            ),
            ElevatedButton(
              onPressed: _isLoading ? null : () => _handleSave(provider),
              style: ElevatedButton.styleFrom(
                backgroundColor: const Color(0xFF8B5CF6),
                foregroundColor: Colors.white,
              ),
              child: _isLoading
                  ? const SizedBox(
                      width: 16,
                      height: 16,
                      child: CircularProgressIndicator(
                        strokeWidth: 2,
                        valueColor: AlwaysStoppedAnimation<Color>(Colors.white),
                      ),
                    )
                  : const Text('Save'),
            ),
          ],
        );
      },
    );
  }
}