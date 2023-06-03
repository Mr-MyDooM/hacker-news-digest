# coding: utf-8
import os
from unittest import TestCase

from page_content_extractor.pdf import *


class PdfParserTestCase(TestCase):

    def test_paragraph_parse_without_authors(self):
        fpath = os.path.join(os.path.dirname(__file__), 'fixtures/cpi.pdf')
        with open(fpath, 'rb') as fp:
            parser = PdfExtractor(fp.read())
            self.assertIsNone(parser.get_illustration())
            self.assertTrue(parser.get_content().startswith(
                'Systems code is often written in low-level languages like C/C++, which offer'
            ))  # Should be no errors

    # def test_text_order(self):
    #     parser = PdfExtractor(open('/tmp/fm_21-76_us_army_survival_manual_2006.pdf', 'rb').read())
    #     self.assertIsNone(parser.get_illustration())
    #     print parser.get_content()
