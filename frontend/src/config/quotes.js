// 首页欢迎语（古文名句，前端内置）
export const QUOTES = [
  { text: '工欲善其事，必先利其器。', source: '《论语·卫灵公》' },
  { text: '博学之，审问之，慎思之，明辨之，笃行之。', source: '《礼记·中庸》' },
  { text: '不积跬步，无以至千里；不积小流，无以成江海。', source: '《荀子·劝学》' },
  { text: '业精于勤，荒于嬉；行成于思，毁于随。', source: '韩愈《进学解》' },
  { text: '纸上得来终觉浅，绝知此事要躬行。', source: '陆游《冬夜读书示子聿》' },
  { text: '路漫漫其修远兮，吾将上下而求索。', source: '屈原《离骚》' },
  { text: '锲而不舍，金石可镂。', source: '《荀子·劝学》' },
  { text: '知之为知之，不知为不知，是知也。', source: '《论语·为政》' },
]

export function randomQuote() {
  return QUOTES[Math.floor(Math.random() * QUOTES.length)]
}
