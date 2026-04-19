import 'package:dio/dio.dart';
import '../models/session.dart';
import '../models/project.dart';
import '../models/llm_config.dart';

class ApiException implements Exception {
  final String message;
  final int? statusCode;
  
  ApiException(this.message, {this.statusCode});
  
  @override
  String toString() => 'ApiException: $message (status: $statusCode)';
}

class ApiService {
  static const String baseUrl = 'http://127.0.0.1:8888';
  final Dio _dio;

  ApiService() : _dio = Dio(BaseOptions(
    baseUrl: baseUrl,
    connectTimeout: const Duration(seconds: 10),
    receiveTimeout: const Duration(seconds: 30),
  ));

  // ============ Project APIs ============

  Future<List<Project>> listProjects() async {
    try {
      final resp = await _dio.get('/api/projects');
      final projectsData = resp.data['projects'];
      if (projectsData == null) return [];
      return (projectsData as List<dynamic>)
          .map((e) => Project.fromJson(e as Map<String, dynamic>))
          .toList();
    } on DioException catch (e) {
      throw ApiException(e.message ?? 'Failed to list projects', statusCode: e.response?.statusCode);
    }
  }

  Future<Project> createProject(String path) async {
    try {
      final resp = await _dio.post('/api/projects', data: {'path': path});
      return Project.fromJson(resp.data);
    } on DioException catch (e) {
      throw ApiException(e.message ?? 'Failed to create project', statusCode: e.response?.statusCode);
    }
  }

  Future<Project> getProject(String name) async {
    try {
      final resp = await _dio.get('/api/projects/$name');
      return Project.fromJson(resp.data);
    } on DioException catch (e) {
      throw ApiException(e.message ?? 'Failed to get project', statusCode: e.response?.statusCode);
    }
  }

  Future<Project> removeSessionFromProject(String projectName, String sessionId) async {
    try {
      final resp = await _dio.delete('/api/projects/$projectName/sessions/$sessionId');
      return Project.fromJson(resp.data);
    } on DioException catch (e) {
      throw ApiException(e.message ?? 'Failed to remove session', statusCode: e.response?.statusCode);
    }
  }

  // ============ Session APIs ============

  Future<List<Session>> listSessions({String? projectName}) async {
    try {
      final resp = await _dio.get('/api/sessions', queryParameters: {
        if (projectName != null) 'project_path': projectName,
      });
      final sessionsData = resp.data['sessions'];
      if (sessionsData == null) return [];
      return (sessionsData as List<dynamic>)
          .map((e) => Session.fromJson(e as Map<String, dynamic>))
          .toList();
    } on DioException catch (e) {
      throw ApiException(e.message ?? 'Failed to list sessions', statusCode: e.response?.statusCode);
    }
  }

  Future<Session> createSession(String projectPath, {String? model}) async {
    try {
      final resp = await _dio.post('/api/sessions', data: {
        'project_path': projectPath,
        if (model != null) 'model': model,
      });
      return Session.fromJson(resp.data);
    } on DioException catch (e) {
      throw ApiException(e.message ?? 'Failed to create session', statusCode: e.response?.statusCode);
    }
  }

  Future<Session> getSession(String id) async {
    try {
      final resp = await _dio.get('/api/sessions/$id');
      return Session.fromJson(resp.data);
    } on DioException catch (e) {
      throw ApiException(e.message ?? 'Failed to get session', statusCode: e.response?.statusCode);
    }
  }

  Future<Session> submitInput(String id, String input, {String? mode}) async {
    try {
      final resp = await _dio.post('/api/sessions/$id/input', data: {
        'input': input,
        if (mode != null) 'mode': mode,
      });
      return Session.fromJson(resp.data);
    } on DioException catch (e) {
      throw ApiException(e.message ?? 'Failed to submit input', statusCode: e.response?.statusCode);
    }
  }

  Future<Session> approve(String id) async {
    try {
      final resp = await _dio.post('/api/sessions/$id/approve');
      return Session.fromJson(resp.data);
    } on DioException catch (e) {
      throw ApiException(e.message ?? 'Failed to approve', statusCode: e.response?.statusCode);
    }
  }

  Future<Session> unblock(String id, bool approved, {String? addAllowed}) async {
    try {
      final resp = await _dio.post('/api/sessions/$id/unblock', data: {
        'approved': approved,
        if (addAllowed != null) 'add_allowed': addAllowed,
      });
      return Session.fromJson(resp.data);
    } on DioException catch (e) {
      throw ApiException(e.message ?? 'Failed to unblock', statusCode: e.response?.statusCode);
    }
  }

  // ============ LLM Config APIs ============

  Future<LLMConfig> getLLMConfig() async {
    try {
      final resp = await _dio.get('/api/config/llm');
      return LLMConfig.fromJson(resp.data);
    } on DioException catch (e) {
      throw ApiException(e.message ?? 'Failed to get LLM config', statusCode: e.response?.statusCode);
    }
  }

  Future<LLMConfig> updateLLMConfig(LLMConfig config) async {
    try {
      final resp = await _dio.put('/api/config/llm', data: config.toJson());
      return LLMConfig.fromJson(resp.data);
    } on DioException catch (e) {
      throw ApiException(e.message ?? 'Failed to update LLM config', statusCode: e.response?.statusCode);
    }
  }
}