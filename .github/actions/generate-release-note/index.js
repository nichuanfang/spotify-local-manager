const core = require('@actions/core');
const github = require('@actions/github');
const fs = require('fs');

async function run() {
    try {
        // 获取提交记录
        const {data: commits} = await github.rest.repos.compareCommits({
            owner: github.context.repo.owner,
            repo: github.context.repo.repo,
            base: github.context.payload.base_ref,
            head: github.context.payload.pull_request.head.sha,
        });

        // 生成发布说明
        let releaseNote = '';
        commits.commits.forEach(commit => {
            const message = commit.commit.message;
            if (message.startsWith('perf:')) {
                releaseNote += `[Performance] ${message.substring(5)}\n`;
            } else if (message.startsWith('fixed:')) {
                releaseNote += `[Fix] ${message.substring(6)}\n`;
            } else if (message.startsWith('feat:')) {
                releaseNote += `[Feature] ${message.substring(5)}\n`;
            }
        });

        // 将发布说明写入文件
        const releaseNoteFile = core.getInput('release_note_file');
        fs.writeFileSync(releaseNoteFile, releaseNote);

        core.setOutput('release_note', releaseNote);
    } catch (error) {
        core.setFailed(error.message);
    }
}

run();
