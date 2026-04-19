import 'dart:async';
import 'dart:convert';
import 'package:dio/dio.dart';
import '../models/event.dart';

class SseService {
  static const String baseUrl = 'http://127.0.0.1:8888';
  final Dio _dio = Dio();

  Stream<SessionEvent> connect(String sessionId) {
    final controller = StreamController<SessionEvent>();
    final url = '$baseUrl/api/sessions/$sessionId/events';

    _dio
        .get(
          url,
          options: Options(
            responseType: ResponseType.stream,
            headers: {'Accept': 'text/event-stream'},
          ),
        )
        .then((response) {
          final responseBody = response.data as ResponseBody;
          final stream = responseBody.stream;
          final buffer = StringBuffer();

          stream.listen(
            (data) {
              buffer.write(String.fromCharCodes(data));
              _parseBuffer(buffer, controller);
            },
            onError: (e) => controller.addError(e),
            onDone: () {
              _parseBuffer(buffer, controller);
              controller.close();
            },
          );
        })
        .catchError((e) {
          controller.addError(e);
          controller.close();
        });

    return controller.stream;
  }

  void _parseBuffer(StringBuffer buffer, StreamController<SessionEvent> controller) {
    final content = buffer.toString();
    buffer.clear();

    if (content.isEmpty) return;

    final lines = content.split('\n');
    for (final line in lines) {
      if (line.startsWith('data: ')) {
        final jsonStr = line.substring(6).trim();
        if (jsonStr.isEmpty) continue;

        try {
          final json = jsonDecode(jsonStr) as Map<String, dynamic>;
          final event = SessionEvent.fromJson(json);
          controller.add(event);
        } catch (e) {
          // JSON 解析失败，忽略此行
        }
      }
    }
  }
}