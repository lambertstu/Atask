class LLMConfig {
  final String apiKey;
  final String baseUrl;
  final String model;
  final List<String> models;
  final bool hasApiKey;
  final String updatedAt;

  LLMConfig({
    required this.apiKey,
    required this.baseUrl,
    required this.model,
    required this.models,
    required this.hasApiKey,
    required this.updatedAt,
  });

  factory LLMConfig.fromJson(Map<String, dynamic> json) {
    return LLMConfig(
      apiKey: json['api_key'] as String? ?? '',
      baseUrl: json['base_url'] as String? ?? '',
      model: json['model'] as String? ?? '',
      models: (json['models'] as List<dynamic>?)?.map((e) => e as String).toList() ?? [],
      hasApiKey: json['has_api_key'] as bool? ?? false,
      updatedAt: json['updated_at'] as String? ?? '',
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'api_key': apiKey,
      'base_url': baseUrl,
      'model': model,
      'models': models,
    };
  }
}