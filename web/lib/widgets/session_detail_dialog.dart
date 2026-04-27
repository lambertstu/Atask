import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../models/session.dart';
import '../models/message.dart';
import '../providers/session_provider.dart';
import '../providers/settings_provider.dart';
import 'message_bubble.dart';

class SessionDetailDialog extends StatefulWidget {
  final String sessionId;

  const SessionDetailDialog({super.key, required this.sessionId});

  @override
  State<SessionDetailDialog> createState() => _SessionDetailDialogState();
}

class _SessionDetailDialogState extends State<SessionDetailDialog> {
  final TextEditingController _inputController = TextEditingController();
  final TextEditingController _addAllowedController = TextEditingController();
  final ScrollController _scrollController = ScrollController();

  late SessionMode _currentMode;
  late String _currentModel;
  bool _isSubmitting = false;
  bool _hasSubscribedToSse = false;
  
  bool _isUserAtBottom = true;
  int _unreadMessageCount = 0;
  int _lastMessageCount = 0;

  @override
  void initState() {
    super.initState();
    final session = _getCurrentSession();
    _currentMode = session?.mode ?? SessionMode.plan;
    _currentModel = session?.model ?? 'glm-5';
    _lastMessageCount = session?.messages.length ?? 0;

    _scrollController.addListener(_onScroll);

    WidgetsBinding.instance.addPostFrameCallback((_) {
      final session = _getCurrentSession();
      if (session != null &&
          (session.state == SessionState.processing ||
           session.state == SessionState.planning ||
           session.state == SessionState.blocked)) {
        context.read<SessionProvider>().subscribeToSession(session.id);
        _hasSubscribedToSse = true;
      }

      if (_scrollController.hasClients) {
        _scrollController.jumpTo(_scrollController.position.maxScrollExtent);
      }
    });
  }

  void _onScroll() {
    if (_scrollController.hasClients) {
      final maxScroll = _scrollController.position.maxScrollExtent;
      final currentScroll = _scrollController.position.pixels;
      final isAtBottom = (maxScroll - currentScroll) < 100;
      
      if (isAtBottom && !_isUserAtBottom) {
        setState(() {
          _isUserAtBottom = true;
          _unreadMessageCount = 0;
        });
      } else if (!isAtBottom && _isUserAtBottom) {
        setState(() {
          _isUserAtBottom = false;
        });
      }
    }
  }

  void _scrollToBottom() {
    if (_scrollController.hasClients) {
      _scrollController.animateTo(
        _scrollController.position.maxScrollExtent,
        duration: const Duration(milliseconds: 300),
        curve: Curves.easeOut,
      );
      setState(() {
        _isUserAtBottom = true;
        _unreadMessageCount = 0;
      });
    }
  }

  @override
  void dispose() {
    _inputController.dispose();
    _addAllowedController.dispose();
    _scrollController.dispose();
    if (_hasSubscribedToSse) {
      context.read<SessionProvider>().unsubscribeFromSession(widget.sessionId);
    }
    super.dispose();
  }

  Session? _getCurrentSession() {
    final provider = context.read<SessionProvider>();
    final sessions = provider.sessions.where((s) => s.id == widget.sessionId).toList();
    return sessions.isEmpty ? null : sessions.first;
  }

  List<_ProcessedMessage> _processMessages(List<ChatMessage> messages) {
    final toolResultsMap = <String, String>{};

    for (final msg in messages) {
      if (msg.role == 'tool' && msg.toolCallId != null && msg.content != null) {
        if (msg.content!.contains('<system-reminder>') || msg.content!.contains('[Previous:')) {
          continue;
        }
        toolResultsMap[msg.toolCallId!] = msg.content!;
      }
    }

    final assistantMessages = messages
        .where((msg) => msg.role != 'tool')
        .map((msg) => _ProcessedMessage(
          message: msg,
          toolResults: toolResultsMap,
        ))
        .toList();

    return _mergeEmptyContentMessages(assistantMessages);
  }

  List<_ProcessedMessage> _mergeEmptyContentMessages(List<_ProcessedMessage> messages) {
    if (messages.isEmpty) return [];

    final result = <_ProcessedMessage>[];
    var i = 0;

    while (i < messages.length) {
      final current = messages[i];
      final hasContent = current.message.content != null && current.message.content!.isNotEmpty;

      if (hasContent) {
        result.add(current);
        i++;
      } else {
        var mergedToolCalls = <ToolCall>[...?current.message.toolCalls];
        var mergedToolResults = Map<String, String>.from(current.toolResults);
        String? mergedReasoning = current.message.reasoningContent;

        i++;

        while (i < messages.length) {
          final next = messages[i];
          final nextHasContent = next.message.content != null && next.message.content!.isNotEmpty;
          if (nextHasContent) break;

          mergedToolCalls.addAll(next.message.toolCalls ?? []);
          mergedToolResults.addAll(next.toolResults);
          if (mergedReasoning == null && next.message.reasoningContent != null) {
            mergedReasoning = next.message.reasoningContent;
          }
          i++;
        }

        if (mergedToolCalls.isNotEmpty || mergedReasoning != null) {
          result.add(_ProcessedMessage(
            message: ChatMessage(
              role: current.message.role,
              content: null,
              reasoningContent: mergedReasoning,
              toolCalls: mergedToolCalls,
            ),
            toolResults: mergedToolResults,
          ));
        }
      }
    }

    return result;
  }

  Future<void> _handleSubmit(SessionProvider provider) async {
    final input = _inputController.text.trim();
    if (input.isEmpty || _isSubmitting) return;

    setState(() {
      _isSubmitting = true;
    });

    try {
      await provider.submitInput(
        widget.sessionId,
        input,
        mode: _currentMode.name,
        model: _currentModel,
      );

      setState(() {
        _inputController.clear();
        _isSubmitting = false;
      });

      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (_scrollController.hasClients) {
          _scrollController.animateTo(
            _scrollController.position.maxScrollExtent,
            duration: const Duration(milliseconds: 300),
            curve: Curves.easeOut,
          );
        }
      });
    } catch (e) {
      if (mounted) {
        setState(() {
          _isSubmitting = false;
        });
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('提交失败：$e')),
        );
      }
    }
  }

  Color _getStateColor(SessionState state) {
    switch (state) {
      case SessionState.pending:
        return const Color(0xFF3B82F6);
      case SessionState.planning:
        return const Color(0xFF8B5CF6);
      case SessionState.processing:
        return const Color(0xFF10B981);
      case SessionState.blocked:
        return const Color(0xFFF59E0B);
      case SessionState.completed:
        return const Color(0xFF059669);
    }
  }

  @override
  Widget build(BuildContext context) {
    return Consumer<SessionProvider>(
      builder: (context, provider, _) {
        final matchedSessions = provider.sessions.where((s) => s.id == widget.sessionId).toList();
        final session = matchedSessions.isEmpty ? null : matchedSessions.first;

        if (session == null) {
          return AlertDialog(
            title: const Text('Session Not Found'),
            content: const Text('The session has been removed.'),
            actions: [
              TextButton(
                onPressed: () => Navigator.pop(context),
                child: const Text('Close'),
              ),
            ],
          );
        }

        final processedMessages = _processMessages(session.messages);
        final isRunning = session.state == SessionState.processing ||
            session.state == SessionState.planning;
        
        final newCount = processedMessages.length;
        if (newCount > _lastMessageCount) {
          final addedCount = newCount - _lastMessageCount;
          if (_isUserAtBottom) {
            WidgetsBinding.instance.addPostFrameCallback((_) {
              _scrollToBottom();
            });
          } else {
            WidgetsBinding.instance.addPostFrameCallback((_) {
              setState(() {
                _unreadMessageCount += addedCount;
              });
            });
          }
          _lastMessageCount = newCount;
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
                  Icons.chat,
                  color: Color(0xFF8B5CF6),
                  size: 24,
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    SelectableText(
                      session.id,
                      style: const TextStyle(
                        fontSize: 14,
                        fontFamily: 'monospace',
                        fontWeight: FontWeight.w500,
                      ),
                    ),
                    const SizedBox(height: 2),
                    Text(
                      session.projectName,
                      style: TextStyle(
                        fontSize: 12,
                        color: Colors.grey[600],
                      ),
                    ),
                  ],
                ),
              ),
              Container(
                padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
                decoration: BoxDecoration(
                  color: _getStateColor(session.state).withAlpha(30),
                  borderRadius: BorderRadius.circular(8),
                  border: Border.all(
                    color: _getStateColor(session.state).withAlpha(80),
                    width: 1,
                  ),
                ),
                child: Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    if (isRunning)
                      Padding(
                        padding: const EdgeInsets.only(right: 8),
                        child: SizedBox(
                          width: 12,
                          height: 12,
                          child: CircularProgressIndicator(
                            strokeWidth: 2,
                            valueColor: AlwaysStoppedAnimation<Color>(
                              _getStateColor(session.state),
                            ),
                          ),
                        ),
                      ),
                    Text(
                      session.state.name.toUpperCase(),
                      style: TextStyle(
                        color: _getStateColor(session.state),
                        fontWeight: FontWeight.w600,
                        fontSize: 12,
                      ),
                    ),
                  ],
                ),
              ),
            ],
          ),
          content: SizedBox(
            width: 900,
            height: 700,
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Expanded(
                  child: processedMessages.isEmpty
                      ? Center(
                          child: Text(
                            'No messages',
                            style: TextStyle(color: Colors.grey[400]),
                          ),
                        )
                      : Stack(
                          children: [
                            ListView.builder(
                              controller: _scrollController,
                              padding: const EdgeInsets.symmetric(vertical: 12),
                              itemCount: processedMessages.length,
                              itemBuilder: (context, index) {
                                final item = processedMessages[index];
                                return MessageBubble(
                                  message: item.message,
                                  toolResults: item.toolResults,
                                );
                              },
                            ),
                            if (_unreadMessageCount > 0)
                              Positioned(
                                top: 16,
                                left: 0,
                                right: 0,
                                child: Center(
                                  child: GestureDetector(
                                    onTap: _scrollToBottom,
                                    child: Container(
                                      padding: const EdgeInsets.symmetric(
                                        horizontal: 16,
                                        vertical: 8,
                                      ),
                                      decoration: BoxDecoration(
                                        color: const Color(0xFF8B5CF6),
                                        borderRadius: BorderRadius.circular(20),
                                      ),
                                      child: Text(
                                        '$_unreadMessageCount 条新消息',
                                        style: const TextStyle(
                                          color: Colors.white,
                                          fontSize: 13,
                                          fontWeight: FontWeight.w500,
                                        ),
                                      ),
                                    ),
                                  ),
                                ),
                              ),
                            if (!_isUserAtBottom && _scrollController.hasClients && 
                                _scrollController.position.maxScrollExtent > 0)
                              Positioned(
                                bottom: 16,
                                left: 0,
                                right: 0,
                                child: Center(
                                  child: GestureDetector(
                                    onTap: _scrollToBottom,
                                    child: Container(
                                      padding: const EdgeInsets.all(10),
                                      decoration: BoxDecoration(
                                        color: Colors.white,
                                        borderRadius: BorderRadius.circular(24),
                                        boxShadow: [
                                          BoxShadow(
                                            color: Colors.black.withAlpha(30),
                                            blurRadius: 8,
                                            offset: const Offset(0, 2),
                                          ),
                                        ],
                                      ),
                                      child: const Icon(
                                        Icons.arrow_downward,
                                        size: 20,
                                        color: Color(0xFF8B5CF6),
                                      ),
                                    ),
                                  ),
                                ),
                              ),
                          ],
                        ),
                ),
                const SizedBox(height: 12),
                session.state == SessionState.blocked
                    ? _buildBlockedPanel(provider, session)
                    : _buildInputArea(provider, session),
              ],
            ),
          ),
        );
      },
    );
  }

  String _getBlockedInfo(Session session) {
    final args = session.blockedArgs;
    if (args == null) return '';

    if (args.containsKey('file_path')) {
      return args['file_path'] as String;
    }
    if (args.containsKey('path')) {
      return args['path'] as String;
    }
    if (args.containsKey('command')) {
      return args['command'] as String;
    }
    return '';
  }

  Widget _buildBlockedPanel(SessionProvider provider, Session session) {
    final toolName = session.blockedTool ?? 'Unknown tool';
    final info = _getBlockedInfo(session);

    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: const Color(0xFFF59E0B).withAlpha(25),
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: const Color(0xFFF59E0B), width: 2),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(Icons.warning_amber_rounded, color: const Color(0xFFF59E0B), size: 24),
              const SizedBox(width: 12),
              const Text(
                'Permission Request',
                style: TextStyle(
                  fontWeight: FontWeight.bold,
                  fontSize: 16,
                  color: Color(0xFFF59E0B),
                ),
              ),
            ],
          ),
          const SizedBox(height: 16),
          Container(
            padding: const EdgeInsets.all(12),
            decoration: BoxDecoration(
              color: Colors.white,
              borderRadius: BorderRadius.circular(8),
              border: Border.all(color: Colors.grey[300]!),
            ),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  children: [
                    Icon(Icons.build, size: 16, color: Colors.grey[700]),
                    const SizedBox(width: 8),
                    Text(
                      'Tool: $toolName',
                      style: TextStyle(fontSize: 14, fontWeight: FontWeight.w500),
                    ),
                  ],
                ),
                if (info.isNotEmpty) ...[
                  const SizedBox(height: 8),
                  Row(
                    children: [
                      Icon(Icons.folder, size: 16, color: Colors.grey[700]),
                      const SizedBox(width: 8),
                      Expanded(
                        child: Text(
                          info,
                          style: TextStyle(fontSize: 13, color: Colors.grey[600]),
                          overflow: TextOverflow.ellipsis,
                        ),
                      ),
                    ],
                  ),
                ],
              ],
            ),
          ),
          const SizedBox(height: 16),
          const Text(
            'Add to allowed paths (optional):',
            style: TextStyle(fontSize: 13, fontWeight: FontWeight.w500),
          ),
          const SizedBox(height: 8),
          TextField(
            controller: _addAllowedController,
            decoration: InputDecoration(
              hintText: 'e.g. /Users/.../project',
              hintStyle: TextStyle(color: Colors.grey[400]),
              border: OutlineInputBorder(
                borderRadius: BorderRadius.circular(8),
              ),
              contentPadding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
            ),
          ),
          const SizedBox(height: 16),
          Row(
            children: [
              Expanded(
                child: ElevatedButton.icon(
                  icon: const Icon(Icons.check, size: 18),
                  label: const Text('Approve'),
                  style: ElevatedButton.styleFrom(
                    backgroundColor: Colors.green,
                    foregroundColor: Colors.white,
                    padding: const EdgeInsets.symmetric(vertical: 12),
                    shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(8),
                    ),
                  ),
                  onPressed: () async {
                    await provider.unblock(
                      session.id,
                      true,
                      addAllowed: _addAllowedController.text.trim(),
                    );
                    _addAllowedController.clear();
                  },
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: OutlinedButton.icon(
                  icon: const Icon(Icons.close, size: 18),
                  label: const Text('Reject'),
                  style: OutlinedButton.styleFrom(
                    foregroundColor: Colors.red,
                    side: const BorderSide(color: Colors.red),
                    padding: const EdgeInsets.symmetric(vertical: 12),
                    shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(8),
                    ),
                  ),
                  onPressed: () async {
                    await provider.unblock(session.id, false);
                    _addAllowedController.clear();
                  },
                ),
              ),
              const SizedBox(width: 12),
              TextButton(
                onPressed: () => Navigator.pop(context),
                child: const Text('Close'),
              ),
            ],
          ),
        ],
      ),
    );
  }

  Widget _buildInputArea(SessionProvider provider, Session session) {
    final isRunning = session.state == SessionState.processing ||
        session.state == SessionState.planning;

    final settingsProvider = context.watch<SettingsProvider>();
    final List<String> availableModels = settingsProvider.config?.models.isNotEmpty == true
        ? settingsProvider.config!.models
        : ['glm-5'];

    String displayModel = _currentModel;
    if (!availableModels.contains(displayModel)) {
      displayModel = availableModels.first;
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (mounted && _currentModel != displayModel) {
          setState(() {
            _currentModel = displayModel;
          });
        }
      });
    }

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Row(
          crossAxisAlignment: CrossAxisAlignment.end,
          children: [
            Expanded(
              child: Container(
                decoration: BoxDecoration(
                  color: Colors.white,
                  borderRadius: BorderRadius.circular(12),
                  border: Border.all(color: Colors.grey[300]!),
                ),
                child: TextField(
                  controller: _inputController,
                  decoration: const InputDecoration(
                    hintText: '随便问点什么...我来帮你规划完成',
                    border: InputBorder.none,
                    contentPadding: EdgeInsets.symmetric(
                      horizontal: 16,
                      vertical: 12,
                    ),
                  ),
                  maxLines: 3,
                  minLines: 1,
                  enabled: !_isSubmitting && !isRunning,
                  onSubmitted: (_) => _handleSubmit(provider),
                ),
              ),
            ),
            const SizedBox(width: 12),
            ElevatedButton(
              onPressed: (_isSubmitting || isRunning) ? null : () => _handleSubmit(provider),
              style: ElevatedButton.styleFrom(
                backgroundColor: const Color(0xFF8B5CF6),
                foregroundColor: Colors.white,
                minimumSize: Size.zero,
                tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                padding: const EdgeInsets.symmetric(
                  horizontal: 24,
                  vertical: 16,
                ),
                shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(12),
                ),
              ),
              child: _isSubmitting
                  ? const SizedBox(
                      width: 20,
                      height: 20,
                      child: CircularProgressIndicator(
                        strokeWidth: 2,
                        valueColor: AlwaysStoppedAnimation<Color>(Colors.white),
                      ),
                    )
                  : const Text('提交'),
            ),
          ],
        ),
        const SizedBox(height: 12),
        Row(
          children: [
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
              decoration: BoxDecoration(
                color: Colors.grey[100],
                borderRadius: BorderRadius.circular(8),
                border: Border.all(color: Colors.grey[300]!),
              ),
              child: DropdownButton<SessionMode>(
                value: _currentMode,
                underline: const SizedBox(),
                icon: const Icon(Icons.arrow_drop_down, size: 18),
                items: SessionMode.values.map((mode) {
                  return DropdownMenuItem(
                    value: mode,
                    child: Text(
                      mode.name.toUpperCase(),
                      style: const TextStyle(
                        fontSize: 12,
                        fontWeight: FontWeight.w500,
                      ),
                    ),
                  );
                }).toList(),
                onChanged: (_isSubmitting || isRunning)
                    ? null
                    : (value) {
                        if (value != null) {
                          setState(() {
                            _currentMode = value;
                          });
                        }
                      },
              ),
            ),
            const SizedBox(width: 8),
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
              decoration: BoxDecoration(
                color: Colors.grey[100],
                borderRadius: BorderRadius.circular(8),
                border: Border.all(color: Colors.grey[300]!),
              ),
              child: DropdownButton<String>(
                value: displayModel,
                underline: const SizedBox(),
                icon: const Icon(Icons.arrow_drop_down, size: 18),
                items: availableModels.map((model) {
                  return DropdownMenuItem(
                    value: model,
                    child: Text(model.toUpperCase(), style: const TextStyle(fontSize: 12)),
                  );
                }).toList(),
                onChanged: (_isSubmitting || isRunning)
                    ? null
                    : (value) {
                        if (value != null) {
                          setState(() {
                            _currentModel = value;
                          });
                        }
                      },
              ),
            ),
            const SizedBox(width: 8),
            IconButton(
              icon: const Icon(Icons.delete_outline, size: 20),
              tooltip: 'Delete Session',
              style: IconButton.styleFrom(
                foregroundColor: Colors.red[400],
                minimumSize: const Size(32, 32),
              ),
              onPressed: (_isSubmitting || isRunning)
                  ? null
                  : () async {
                      final confirmed = await showDialog<bool>(
                        context: context,
                        builder: (ctx) => AlertDialog(
                          title: const Text('Delete Session'),
                          content: const Text(
                              'Are you sure you want to delete this session?'),
                          actions: [
                            TextButton(
                              onPressed: () => Navigator.pop(ctx, false),
                              child: const Text('Cancel'),
                            ),
                            TextButton(
                              onPressed: () => Navigator.pop(ctx, true),
                              style: TextButton.styleFrom(
                                  foregroundColor: Colors.red),
                              child: const Text('Delete'),
                            ),
                          ],
                        ),
                      );
                      if (confirmed == true) {
                        await provider.removeSession(session.id);
                        if (mounted) Navigator.pop(context);
                      }
                    },
            ),
            const Spacer(),
            TextButton(
              onPressed: () => Navigator.pop(context),
              child: const Text('Close'),
            ),
          ],
        ),
      ],
    );
  }
}

class _ProcessedMessage {
  final ChatMessage message;
  final Map<String, String> toolResults;

  _ProcessedMessage({
    required this.message,
    required this.toolResults,
  });
}